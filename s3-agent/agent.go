package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const version = "0.0.1"

type S3Agent struct {

}

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
        // Restart itself
		if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
			log.Println(err)
		}
	}
}

func main() {
	doSelfUpdate()


    config := NewConfigPath(nil)
    DBSanitize(config)

    fs := NewS3FS("./tmp", "./hello", config)
    fs.Run(false)

	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
