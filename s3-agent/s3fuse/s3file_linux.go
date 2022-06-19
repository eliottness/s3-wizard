// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s3fuse

import (
	"context"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func (f *S3File) Allocate(ctx context.Context, off uint64, sz uint64, mode uint32) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()

    // TODO
    // if isremote && isfile, download the file and update the db about the file location + delete on the S3

	err := syscall.Fallocate(f.fd, mode, int64(off), int64(sz))
	if err != nil {
		return fs.ToErrno(err)
	}
	return fs.OK
}

// Utimens - file handle based version of S3FileSystem.Utimens()
func (f *S3File) utimens(a *time.Time, m *time.Time) syscall.Errno {
	var ts [2]syscall.Timespec
	ts[0] = fuse.UtimeToTimespec(a)
	ts[1] = fuse.UtimeToTimespec(m)
	err := futimens(int(f.fd), &ts)
	return fs.ToErrno(err)
}

func setBlocks(out *fuse.Attr) {
	if out.Blksize > 0 {
		return
	}

	out.Blksize = 4096
	pages := (out.Size + 4095) / 4096
	out.Blocks = pages * 8
}
