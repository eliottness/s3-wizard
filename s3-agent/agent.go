package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
    "log"
    "github.com/eliottness/s3-agent/s3fuse"

	"github.com/alecthomas/kong"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/alecthomas/kong"
    "github.com/hanwen/go-fuse/v2/fs"
)

const version = "0.0.1"

type Context struct {
	Debug bool
}

type ReceiveCmd struct {
	ConfigPath     string `arg:"" name:"configPath" help:"Path to the agent config file." type:"path"`
	SyncFolderPath string `arg:"" name:"syncFolderPath" help:"Path to the folder to sync." type:"path"`
}

func (cmd *ReceiveCmd) Run(ctx *Context) error {
	fmt.Println("receive", cmd.ConfigPath, cmd.SyncFolderPath)
	return nil
}

type SendCmd struct {
	ConfigPath     string `arg:"" name:"configPath" help:"Path to the agent config file." type:"path"`
	SyncFolderPath string `arg:"" name:"syncFolderPath" help:"Path to the folder to sync." type:"path"`
}

func (cmd *SendCmd) Run(ctx *Context) error {
	fmt.Println("send", cmd.ConfigPath, cmd.SyncFolderPath)
	return nil
}

var cli struct {
	Debug bool       `help:"Enable debug mode."`
	Rm    ReceiveCmd `cmd:"" name:"receive" help:"Remove files."`
	Send  SendCmd    `cmd:"" name:"send" help:"List paths."`
}

func doSelfUpdate() {
    v := semver.MustParse(version)
    latest, err := selfupdate.UpdateSelf(v, "eliottness/s3-wizard")
    if err != nil {
        log.Println("Binary update failed:", err)
        return
    }
    if latest.Version.Equals(v) {
        log.Println("Current binary is the latest version", version)
    } else {
        log.Println("Successfully updated to version", latest.Version)
        log.Println("Release note:\n", latest.ReleaseNotes)
        if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
            log.Println(err)
        }
    }
}

func main() {
    doSelfUpdate()

    loopbackRoot, err := s3fuse.NewS3Root("./tmp")
	if err != nil {
		log.Fatalf("NewLoopbackRoot(%s): %v\n", "./tmp", err)
	}

    opts := &fs.Options{}

	opts.MountOptions.Options = append(opts.MountOptions.Options, "default_permissions")
	// First column in "df -T": original dir
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname=hello")
	// Second column in "df -T" will be shown as "fuse." + Name
	opts.MountOptions.Name = "loopback"
	// Leave file permissions on "000" files as-is
	opts.NullPermissions = true

    opts.MountOptions.EnableLocks = false;
    opts.MountOptions.Debug = true;

	server, err := fs.Mount("./hello", loopbackRoot, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}

    server.Wait()

	ctx := kong.Parse(&cli)
	err = ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
