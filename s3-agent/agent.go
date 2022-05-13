package main

import "github.com/alecthomas/kong"

type Context struct {
    Debug bool
}

type SendCmd struct {
    configPath string `arg:"config" name:"Config Path" help:"Config of the agent." type:"path"`
    syncFolder string `arg:"folder" name:"Sync Folder" help:"Folder to send." type:"path"`
}

func (obj *SendCmd) Run(ctx *Context) error {
    return nil
}

type ReceiveCmd struct {
    configPath string `arg:"config" name:"Config Path" help:"Config of the agent." type:"path"`
    syncFolder string `arg:"folder" name:"Sync Folder" help:"Folder to send." type:"path"`
}

func (obj *ReceiveCmd) Run(ctx *Context) error {
    return nil
}

var CLI struct {
    Debug   bool       `help:"Enable debug mode."`
    Send    SendCmd    `cmd:"send" help:"SendCmd files to the configured remote."`
    Receive ReceiveCmd `cmd:"receive" help:"Retrieve files for the configured remote."`
}

func main() {
    ctx := kong.Parse(&CLI)
    err := ctx.Run(&Context{Debug: CLI.Debug})
    ctx.FatalIfErrorf(err)
}
