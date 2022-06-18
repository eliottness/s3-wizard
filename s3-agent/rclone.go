package main

// #include <stdlib.h>
import "C"

import (
	_ "embed"
	"log"
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

    // never hit
    log.Println("should never hit")
    return err
}

/// Run the rclone binary with the given arguments.
/// The rclone binary is embed in our main binary.
/// This function create a memory space associated with a file descriptor.
/// This file descriptor is passed to execvp with the arguments to run rclone
func RunRClone(args []string) error {
    fd, err := memfdCreate("/file.bin")
    if err != nil {
        return err
    }

    if err = copyToMem(fd, rcloneBinary); err != nil {
        return err
    }

    if err = execveAt(fd, args); err != nil {
        return err
    }

    return nil
}

