package main

// #include <stdlib.h>
import "C"

import (
	_ "embed"
	"log"
	"path"
	"path/filepath"

	"github.com/estebangarcia21/subprocess"
)

type RClone struct {
	config *ConfigPath
}

func NewRClone(config *ConfigPath) (*RClone, error) {

	return &RClone{config: config}, nil
}

/// Run the rclone binary with the given arguments.
/// The rclone binary is embed in our main binary.
/// This function create a memory space associated with a file descriptor.
/// It copies the rclone binary to the memory space.
/// This file descriptor is passed to execvp with the arguments to run rclone
func (r *RClone) Run(opts ...subprocess.Option) (int, error) {

    opts = append(opts, subprocess.Args("--config", r.config.GetRClonePath()))

    pop := subprocess.New("./rclone", opts...)

    if err := pop.Exec(); err != nil {
        return -1, err
    }

    return pop.ExitCode(), nil
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

	ret, err := r.Run(subprocess.Args("copy", entry.Path, entry.Server + ":" + s3Path))
    if ret != 0 {
        log.Println("Rclone send failed with exit code: ", ret)
    }
	return err
}

func (r *RClone) Download(entry *S3NodeTable) error {
	s3Path, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	ret, err := r.Run(subprocess.Args("move", entry.Server + ":" + s3Path + "/" + filepath.Base(entry.Path), path.Dir(entry.Path)))
    if ret != 0 {
        log.Println("Rclone download failed with exit code: ", ret)
    }
	return err
}

func (r *RClone) Remove(entry *S3NodeTable) error {
	s3Path, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	ret, err := r.Run(subprocess.Args("deletefile", entry.Server + ":" + s3Path))
    if ret != 0 {
        log.Println("Rclone remove failed with exit code: ", ret)
    }
	return err
}
