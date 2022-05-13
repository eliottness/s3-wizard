package main

import (
	"fmt"
	"github.com/alecthomas/kong"
)

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

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
