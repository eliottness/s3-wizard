//go:build linux
// +build linux

// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"golang.org/x/sys/unix"
)

func (n *S3Node) Getxattr(ctx context.Context, attr string, dest []byte) (uint32, syscall.Errno) {
	sz, err := unix.Lgetxattr(n.path(), attr, dest)
	return uint32(sz), fs.ToErrno(err)
}

func (n *S3Node) Setxattr(ctx context.Context, attr string, data []byte, flags uint32) syscall.Errno {
	err := unix.Lsetxattr(n.path(), attr, data, int(flags))
	return fs.ToErrno(err)
}

func (n *S3Node) Removexattr(ctx context.Context, attr string) syscall.Errno {
	err := unix.Lremovexattr(n.path(), attr)
	return fs.ToErrno(err)
}

func (n *S3Node) Listxattr(ctx context.Context, dest []byte) (uint32, syscall.Errno) {
	sz, err := unix.Llistxattr(n.path(), dest)
	return uint32(sz), fs.ToErrno(err)
}

func (n *S3Node) renameExchange(name string, newparent fs.InodeEmbedder, newName string) syscall.Errno {
	fd1, err := syscall.Open(n.path(), syscall.O_DIRECTORY, 0)
	if err != nil {
		return fs.ToErrno(err)
	}
	defer syscall.Close(fd1)
	p2 := filepath.Join(n.RootData.Path, newparent.EmbeddedInode().Path(nil))
	fd2, err := syscall.Open(p2, syscall.O_DIRECTORY, 0)
	defer syscall.Close(fd2)
	if err != nil {
		return fs.ToErrno(err)
	}

	var st syscall.Stat_t
	if err := syscall.Fstat(fd1, &st); err != nil {
		return fs.ToErrno(err)
	}

	// Double check that nodes didn't change from under us.
	inode := &n.Inode
	if inode.Root() != inode && inode.StableAttr().Ino != n.RootData.idFromStat(&st).Ino {
		return syscall.EBUSY
	}
	if err := syscall.Fstat(fd2, &st); err != nil {
		return fs.ToErrno(err)
	}

	newinode := newparent.EmbeddedInode()
	if newinode.Root() != newinode && newinode.StableAttr().Ino != n.RootData.idFromStat(&st).Ino {
		return syscall.EBUSY
	}

    if err := n.RootData.fs.Rename(filepath.Join(n.path(), name), filepath.Join(p2, newName)); err != nil {
        return fs.ToErrno(err)
    }

	return fs.ToErrno(unix.Renameat2(fd1, name, fd2, newName, unix.RENAME_EXCHANGE))
}

func (n *S3Node) CopyFileRange(ctx context.Context, fhIn fs.FileHandle,
	offIn uint64, out *fs.Inode, fhOut fs.FileHandle, offOut uint64,
	len uint64, flags uint64) (uint32, syscall.Errno) {


	lfIn, ok := fhIn.(*S3File)
	if !ok {
		return 0, syscall.ENOTSUP
	}
	lfOut, ok := fhOut.(*S3File)
	if !ok {
		return 0, syscall.ENOTSUP
	}

    if err := n.RootData.fs.Download(lfIn.Path); err != nil {
        return 0, fs.ToErrno(err)
    }

    if err := n.RootData.fs.Download(lfOut.Path); err != nil {
        return 0, fs.ToErrno(err)
    }

	signedOffIn := int64(offIn)
	signedOffOut := int64(offOut)
	count, err := unix.CopyFileRange(lfIn.Fd, &signedOffIn, lfOut.Fd, &signedOffOut, int(len), int(flags))
	return uint32(count), fs.ToErrno(err)
}
