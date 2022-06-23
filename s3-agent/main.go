package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/robfig/cron"
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
		log.Println("Cannot load config", err)
		return err
	}

	ctx.ConfigPath.WriteRCloneConfig(config.RCloneConfig)

	rule := config.Rules[0]
	db := Open(ctx.ConfigPath)
	dbEntry := AddIfNotExistsRule(db, rule.Src)
	loopback := ctx.ConfigPath.GetLoopbackFSPath(dbEntry.UUID)

	if _, err := os.Stat(rule.Src); os.IsExist(err) {

		if err := importFS(rule, ctx.ConfigPath); err != nil {
			return err
		}

		if _, err := os.Stat(rule.Src); os.IsExist(err) {
			log.Println("Cannot mount destination: file exists: ", rule.Src)
		}
	}

	fs := NewS3FS(loopback, rule.Src, ctx.ConfigPath)
	sender, err := NewS3Sender(&rule, fs, config.ExcludePatterns, ctx.ConfigPath)
	if err != nil {
		log.Println("Failed to create Cron sender", err)
		return err
	}

	cron := cron.New()
	cron.AddFunc(rule.CronSender, sender.Cycle)
	cron.Start()
	
	if err := fs.Run(ctx.ConfigPath.debug); err != nil {
		log.Printf("Cannot mount filesystem at pas %v", err )
	}

	cron.Stop()
	return nil
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

	for _, rule := range config.Rules {
		if _, err := os.Stat(rule.Src); os.IsExist(err) {
			// A folder exists, import it into the loopback folder and in the DB
			importFS(rule, ctx.ConfigPath)
		}
	}

	return SaveConfig(ctx.ConfigPath.GetAgentConfigPath(), config)
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
	cli := &CLI{
		Debug:        false,
		ConfigFolder: "",
	}
	ctx := kong.Parse(cli)
	err := ctx.Run(&Context{ConfigPath: NewConfigPath(&cli.ConfigFolder, cli.Debug)})
	ctx.FatalIfErrorf(err)
}
