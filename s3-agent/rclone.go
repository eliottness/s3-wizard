package main

// #include <stdlib.h>
import "C"

import (
	_ "embed"
	"log"
	"path/filepath"
	"syscall"
	"unsafe"
)

//go:embed rclone
var rcloneBinary []byte

func memfdCreate(path string) (r1 uintptr, err error) {
	s, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}

	// https://blog.rchapman.org/posts/Linux_System_Call_Table_for_x86_64/
	r1, _, errno := syscall.Syscall(319, uintptr(unsafe.Pointer(s)), 0, 0)

	if int(r1) == -1 {
		return r1, errno
	}

	return r1, nil
}

func copyToMem(fd uintptr, buf []byte) (err error) {
	_, err = syscall.Write(int(fd), buf)
	if err != nil {
		return err
	}

	return nil
}

func execveAt(fd uintptr, args []string) (err error) {

	argv := make([]*C.char, len(args))
	for i, s := range args {
		cs := C.CString(s)
		defer C.free(unsafe.Pointer(cs))
		argv[i] = cs
	}

	s, err := syscall.BytePtrFromString("")
	if err != nil {
		return err
	}
	ret, _, errno := syscall.Syscall6(322, fd, uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&argv[0])), 0, 0x1000, 0)
	if int(ret) == -1 {
		return errno
	}

	// should never hit
	log.Println("Unreachable statement in execveAt")
	return err
}

type RClone struct {
	fd     uintptr
	config *ConfigPath
}

func NewRClone(config *ConfigPath) (*RClone, error) {

	fd, err := memfdCreate("/rclone")
	if err != nil {
		return nil, err
	}

	if err = copyToMem(fd, rcloneBinary); err != nil {
		return nil, err
	}

	return &RClone{fd: fd, config: config}, nil
}

/// Run the rclone binary with the given arguments.
/// The rclone binary is embed in our main binary.
/// This function create a memory space associated with a file descriptor.
/// It copies the rclone binary to the memory space.
/// This file descriptor is passed to execvp with the arguments to run rclone
func (r *RClone) Run(args []string) (int, error) {

	syscall.ForkLock.Lock()

	id, _, _ := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if id != 0 {
		syscall.ForkLock.Unlock()
		var wstatus syscall.WaitStatus
		if _, err := syscall.Wait4(int(id), &wstatus, 0, nil); err != nil {
			return -1, err
		}

		return int(wstatus.ExitStatus()), nil
	}

	// Add the config path
	args = append(args, "--config", r.config.GetRClonePath())
	if err := execveAt(r.fd, args); err != nil {
		return -1, err
	}

	// Should never reach
	log.Println("Unreachable statement in Rclone.Run")
	return -1, nil
}

func (r *RClone) getS3Path(entry *S3NodeTable) (string, error) {
	config, err := LoadConfig(r.config.GetAgentConfigPath())
	if err != nil {
		return "", err
	}

	bucket := config.RCloneConfig[entry.Server]["bucket"]

	return filepath.Join(bucket, "s3-agent", entry.Rule.UUID, entry.UUID), err
}

func (r *RClone) Send(entry *S3NodeTable) error {
	s3Path, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	_, err = r.Run([]string{"./rclone", "copy", entry.Path, entry.Server + ":" + s3Path})
	return err
}

func (r *RClone) Download(entry *S3NodeTable) error {
	_, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	return nil
	// _, err = r.Run([]string{"./rclone", "move", entry.Server + ":" + s3Path, entry.Path})
	// return err
}

func (r *RClone) Remove(entry *S3NodeTable) error {
	s3Path, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	_, err = r.Run([]string{"./rclone", "deletefile", entry.Server + ":" + s3Path})
	return err
}
