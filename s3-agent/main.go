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

type Context struct {
	ConfigPath *ConfigPath
}

type SendCmd struct {
	ConfigPath     string `arg:"" name:"configPath" help:"Path to the agent config file." type:"path"`
	SyncFolderPath string `arg:"" name:"syncFolderPath" help:"Path to the folder to sync." type:"path"`
}

type SyncCmd struct {
	LoopbackPath string `help:"Path to underlying filesystem." type:"path"`
}

func (cmd *SyncCmd) Run(ctx *Context) error {

	doSelfUpdate()

	DBSanitize(ctx.ConfigPath)

	config, err := LoadConfig(ctx.ConfigPath.GetAgentConfigPath())
	if err != nil {
		return err
	}

	ctx.ConfigPath.WriteRCloneConfig(config.RCloneConfig)

	rule := config.Rules[0]
	db := Open(ctx.ConfigPath)
	dbEntry := AddIfNotExistsRule(db, rule.Src)
	loopback := ctx.ConfigPath.GetLoopbackFSPath(dbEntry.UUID)

    if !IsDirectory(rule.Src) {
        if err := os.Mkdir(rule.Src, 755); err != nil {
            return err
        }
    }

	fs := NewS3FS(loopback, rule.Src, ctx.ConfigPath)
	return fs.Run(ctx.ConfigPath.debug)
}

type ConfigCmd struct {
	Import ImportConfigCmd `cmd:"" name:"import" help:"Import the config."`
}

type ImportConfigCmd struct {
	ConfigPath string `arg:"" name:"configPath" help:"Path to the agent config file." type:"path"`
}

func (cmd *ImportConfigCmd) Run(ctx *Context) error {
	fmt.Println("Import", cmd.ConfigPath)

	config, err := LoadConfig(cmd.ConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	err = SaveConfig(ctx.ConfigPath.GetAgentConfigPath(), config)
	return nil
}

type CLI struct {
	Debug        bool      `help:"Enable debug mode."`
	ConfigFolder string    `help:"Path to the agent config folder."`
	Sync         SyncCmd   `cmd:"" name:"sync" help:"Run the sync daemon."`
	Config       ConfigCmd `cmd:"" name:"config" help:"Manage the config."`
}

func doSelfUpdate() {
	v := semver.MustParse(version)
	latest, err := selfupdate.UpdateSelf(v, "eliottness/s3-wizard")
	if err != nil {
		log.Println("s3-agent update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		log.Println("Current s3-agent is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		log.Println("Release note:\n", latest.ReleaseNotes)
		log.Println("Restarting...")
		// Restart itself
		if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
			log.Println(err)
		}
	}
}

func main() {

    //path := "/home/leiyks/Documents/projects/s3-wizard/config.json"
    //configPath := NewConfigPath(&path, true)
    //r, _ := NewRClone(configPath)

    //entry := S3NodeTable{
	//	Path:    path,
	//	Size:    size,
	//	IsLocal: true,
	//	UUID:    uuid.New().String(),
	//	Server:  "",
	//}
    //r.send()

    return

    cli := &CLI{
        Debug: false,
        ConfigFolder: "",
    }
	ctx := kong.Parse(cli)
	err := ctx.Run(&Context{ConfigPath: NewConfigPath(&cli.ConfigFolder, cli.Debug)})
	ctx.FatalIfErrorf(err)
}
