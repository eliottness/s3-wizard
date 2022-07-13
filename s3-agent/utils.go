package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// futimens - futimens(3) calls utimensat(2) with "pathname" set to null and
// "flags" set to zero
func futimens(fd int, times *[2]syscall.Timespec) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(fd), 0, uintptr(unsafe.Pointer(times)), uintptr(0), 0, 0)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}

func IsDirectory(path string) bool {
	fo, err := os.Stat(path)
	return err == nil && fo.IsDir()
}

func IsRegFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		log.Printf("Error statting file: %v", err)
		return false
	}

	return stat.Mode().IsRegular()
}

func IsSubpath(path, subpath string, out *string) bool {
	relativePath, err := filepath.Rel(path, subpath)
	if err == nil && !strings.HasPrefix(relativePath, "..") {
		if out != nil {
			*out = relativePath
		}
		return true
	}
	return false
}
