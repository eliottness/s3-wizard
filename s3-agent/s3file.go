// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"sync"

	//	"time"

	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"golang.org/x/sys/unix"
)

// NewLoopbackFile creates a fs.FileHandle out of a file descriptor. All
// operations are implemented. When using the Fd from a *os.File, call
// syscall.Dup() on the fd, to avoid os.File's finalizer from closing
// the file descriptor.
func NewS3File(root *S3Root, fd int, path string, flags uint32) *S3File {
	return &S3File{root: root, Fd: fd, Path: path, Flags: flags}
}

type S3File struct {
	// Mutex on operations
	Mutex sync.Mutex

	// The root of the filesystem
	root *S3Root

	// The file descriptor (or -1 of there is none)
	Fd int

	// The path of the file
	Path string

	// Opening flags in case we have to reopen it
	Flags uint32
}

var _ = (fs.FileHandle)((*S3File)(nil))
var _ = (fs.FileReleaser)((*S3File)(nil))
var _ = (fs.FileGetattrer)((*S3File)(nil))
var _ = (fs.FileReader)((*S3File)(nil))
var _ = (fs.FileWriter)((*S3File)(nil))
var _ = (fs.FileLseeker)((*S3File)(nil))
var _ = (fs.FileFlusher)((*S3File)(nil))
var _ = (fs.FileFsyncer)((*S3File)(nil))
var _ = (fs.FileSetattrer)((*S3File)(nil))
var _ = (fs.FileAllocater)((*S3File)(nil))

// Removed using the EnableLocks = false in the mount options.
// var _ = (fs.FileGetlker)((*S3File)(nil))
// var _ = (fs.FileSetlker)((*S3File)(nil))
// var _ = (fs.FileSetlkwer)((*S3File)(nil))

func (f *S3File) Read(ctx context.Context, buf []byte, off int64) (res fuse.ReadResult, errno syscall.Errno) {

	// The user asked the real data, we need to download the file or verify the cache
	if err := f.root.fs.Download(f.Path); err != nil {
		return nil, fs.ToErrno(err)
	}

	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	r := fuse.ReadResultFd(uintptr(f.Fd), off, len(buf))
	return r, fs.OK
}

func (f *S3File) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {

	// The user asked the real data, we need to download the file or verify the cache
	if err := f.root.fs.Download(f.Path); err != nil {
		return 0, fs.ToErrno(err)
	}

	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	n, err := syscall.Pwrite(f.Fd, data, off)
	return uint32(n), fs.ToErrno(err)
}

func (f *S3File) Release(ctx context.Context) syscall.Errno {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if f.Fd != -1 {

		if err := f.root.fs.UnregisterFH(f); err != nil {
			return fs.ToErrno(err)
		}

		err := syscall.Close(f.Fd)
		f.Fd = -1
		return fs.ToErrno(err)
	}
	return syscall.EBADF
}

func (f *S3File) Flush(ctx context.Context) syscall.Errno {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(f.Fd)

	if err != nil {
		return fs.ToErrno(err)
	}
	err = syscall.Close(newFd)
	return fs.ToErrno(err)
}

func (f *S3File) Fsync(ctx context.Context, flags uint32) (errno syscall.Errno) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	r := fs.ToErrno(syscall.Fsync(f.Fd))

	return r
}

func (f *S3File) Setattr(ctx context.Context, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if errno := f.setAttr(ctx, in); errno != 0 {
		return errno
	}

	return f.Getattr(ctx, out)
}

func (f *S3File) setAttr(ctx context.Context, in *fuse.SetAttrIn) syscall.Errno {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	var errno syscall.Errno
	if mode, ok := in.GetMode(); ok {
		errno = fs.ToErrno(syscall.Fchmod(f.Fd, mode))
		if errno != 0 {
			return errno
		}
	}

	uid32, uOk := in.GetUID()
	gid32, gOk := in.GetGID()
	if uOk || gOk {
		uid := -1
		gid := -1

		if uOk {
			uid = int(uid32)
		}
		if gOk {
			gid = int(gid32)
		}
		errno = fs.ToErrno(syscall.Fchown(f.Fd, uid, gid))
		if errno != 0 {
			return errno
		}
	}

	mtime, mok := in.GetMTime()
	atime, aok := in.GetATime()

	if mok || aok {
		ap := &atime
		mp := &mtime
		if !aok {
			ap = nil
		}
		if !mok {
			mp = nil
		}
		errno = f.utimens(ap, mp)
		if errno != 0 {
			return errno
		}
	}

	if sz, ok := in.GetSize(); ok {

		// The user ask to truncate the file, so we need to download the file.
		if err := f.root.fs.Download(f.Path); err != nil {
			return fs.ToErrno(err)
		}

		errno = fs.ToErrno(syscall.Ftruncate(f.Fd, int64(sz)))
		if errno != 0 {
			return errno
		}
	}
	return fs.OK
}

func (f *S3File) Getattr(ctx context.Context, a *fuse.AttrOut) syscall.Errno {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	st := syscall.Stat_t{}
	err := syscall.Fstat(f.Fd, &st)
	if err != nil {
		return fs.ToErrno(err)
	}

	// We have to get the fake size of the file
	size, err := f.root.fs.GetSize(f.Path)
	if err != nil {
		return fs.ToErrno(err)
	}

	st.Size = size

	a.FromStat(&st)

	return fs.OK
}

func (f *S3File) Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno) {
	if err := f.root.fs.Download(f.Path); err != nil {
		return 0, fs.ToErrno(err)
	}

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	n, err := unix.Seek(f.Fd, int64(off), int(whence))
	return uint64(n), fs.ToErrno(err)
}
