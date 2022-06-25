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
	logger     *log.Logger
}

func NewRClone(configPath *ConfigPath) *RClone {
	rclonePath := configPath.GetRCloneBinaryPath()
	file, err := os.Create(rclonePath)
	if err != nil {
		panic(err)
	}

	file.Write(rcloneBinary)
	file.Close()
	os.Chmod(rclonePath, 0700)

	config, err := LoadConfig(configPath.GetAgentConfigPath())
	if err != nil {
		panic(err)
	}

	return &RClone{configPath: configPath, config: config, logger: configPath.NewLogger("RCLONE: ")}
}

/// Run the rclone binary with the given arguments.
/// The rclone binary is embed in our main binary.
/// This function create a memory space associated with a file descriptor.
/// It copies the rclone binary to the memory space.
/// This file descriptor is passed to execvp with the arguments to run rclone
func (r *RClone) Run(opts ...subprocess.Option) (int, error) {

	opts = append(opts, subprocess.Args("--config", r.configPath.GetRCloneConfigPath()))
	pop := subprocess.New(r.configPath.GetRCloneBinaryPath(), opts...)

	if err := pop.Exec(); err != nil {
		return -1, err
	}

	return pop.ExitCode(), nil
}

func (r *RClone) getS3Path(server, ruleId, entryId string) string {

	bucket := r.config.RCloneConfig[server]["bucket"]
	serverPath := filepath.Join(bucket, "s3-agent", ruleId, entryId)

	return server + ":" + serverPath
}

func (r *RClone) Send(server string, entry *S3NodeTable) error {
	if !entry.Local {
		r.logger.Println("Warning: Asking RClone to send a remote file")
		return nil
	}

	s3Path := r.getS3Path(server, entry.S3RuleTable.UUID, entry.UUID)

	ret, err := r.Run(subprocess.Args("copyto", entry.Path, s3Path))
	if ret != 0 {
		r.logger.Println("Rclone send failed with exit code: ", ret)
		return err
	}

	return nil
}

func (r *RClone) Download(entry *S3NodeTable) error {
	if entry.Local {
		r.logger.Println("Warning: Asking RClone to download a local file")
		return nil
	}

	s3Path := r.getS3Path(entry.Server, entry.S3RuleTable.UUID, entry.UUID)

	ret, err := r.Run(subprocess.Args("moveto", s3Path, entry.Path))
	if ret != 0 {
		r.logger.Println("Rclone download failed with exit code: ", ret)
		return err
	}

	return nil
}

func (r *RClone) Remove(entry *S3NodeTable) error {
	if entry.Local {
		r.logger.Println("Warning: Asking RClone to remove a local file")
		return nil
	}

	s3Path := r.getS3Path(entry.Server, entry.S3RuleTable.UUID, entry.UUID)

	ret, err := r.Run(subprocess.Args("deletefile", s3Path))
	if ret != 0 {
		r.logger.Println("Rclone remove failed with exit code: ", ret)
		return err
	}

	return nil
}
