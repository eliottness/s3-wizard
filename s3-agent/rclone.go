package main

// #include <stdlib.h>
import "C"

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/estebangarcia21/subprocess"
)

//go:embed rclone
var rcloneBinary []byte

type RClone struct {
	config     *Config
	configPath *ConfigPath
}

func NewRClone(configPath *ConfigPath) *RClone {
	rclonePath := configPath.GetRcloneBinaryPath()
	file, err := os.OpenFile(rclonePath, os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}

	defer file.Close()
	file.Write(rcloneBinary)

	config, err := LoadConfig(configPath.GetAgentConfigPath())
	if err != nil {
		panic(err)
	}

	return &RClone{configPath: configPath, config: config}
}

/// Run the rclone binary with the given arguments.
/// The rclone binary is embed in our main binary.
/// This function create a memory space associated with a file descriptor.
/// It copies the rclone binary to the memory space.
/// This file descriptor is passed to execvp with the arguments to run rclone
func (r *RClone) Run(opts ...subprocess.Option) (int, error) {

	opts = append(opts, subprocess.Args("--config", r.configPath.GetRCloneConfigPath()))
	pop := subprocess.New(r.configPath.GetRcloneBinaryPath(), opts...)

	if err := pop.Exec(); err != nil {
		return -1, err
	}

	return pop.ExitCode(), nil
}

func (r *RClone) getS3Path(entry *S3NodeTable) (string, error) {

	bucket := r.config.RCloneConfig[entry.Server]["bucket"]
	serverPath := filepath.Join(bucket, "s3-agent", entry.S3RuleTable.UUID, entry.UUID)

	return entry.Server + ":" + serverPath, nil
}

func (r *RClone) Send(entry *S3NodeTable) error {
	s3Path, err := r.getS3Path(entry)

	if err != nil {
		return err
	}

	ret, err := r.Run(subprocess.Args("copyto", entry.Path, s3Path))
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

	ret, err := r.Run(subprocess.Args("moveto", s3Path, entry.Path))
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

	ret, err := r.Run(subprocess.Args("deletefile", s3Path))
	if ret != 0 {
		log.Println("Rclone remove failed with exit code: ", ret)
	}
	return err
}
