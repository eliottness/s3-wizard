package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/robfig/cron"
)

const version = "0.1.1"

type Context struct {
	ConfigPath *ConfigPath
}

type SyncCmd struct{}

func (cmd *SyncCmd) Run(ctx *Context) error {
	config, err := LoadConfig(ctx.ConfigPath.GetAgentConfigPath())
	if err != nil {
		log.Println("Cannot load config", err)
		return err
	}

	if err = ctx.ConfigPath.WriteRCloneConfig(config.RCloneConfig); err != nil {
		log.Println("Cannot write rclone config", err)
		return err
	}

	rule := config.Rules[0]
	orm := NewSQlite(ctx.ConfigPath)
	dbEntry := orm.AddIfNotExistsRule(rule.Src)
	loopback := ctx.ConfigPath.GetLoopbackFSPath(dbEntry.UUID)

	if _, err := os.Stat(rule.Src); err == nil {

		if err := importFS(rule, ctx.ConfigPath, orm); err != nil {
			log.Println("Error while importing existing files: ", err)
			return err
		}

		if _, err := os.Stat(rule.Src); err == nil {
			log.Println("Cannot mount destination: file exists: ", rule.Src)
		}
	}

	fs := NewS3FS(loopback, rule.Src, ctx.ConfigPath, orm)
	sender, err := NewS3Sender(&rule, fs, config.ExcludePatterns, ctx.ConfigPath, orm)
	if err != nil {
		log.Println("Failed to create Cron sender", err)
		return err
	}

	cron := cron.New()
	cron.AddFunc(rule.CronSender, sender.Cycle)
	cron.Start()

	if err := fs.Run(ctx.ConfigPath.debug); err != nil {
		log.Printf("Cannot mount filesystem at pas %v", err)
		return err
	}

	fs.WaitStop()

	cron.Stop()
	return nil
}

type RebuildDbCmd struct {
	UUID       string `arg:"" help:"UUID of the rule to be used to rebuild the database"`
	RuleNumber int    `arg:"" help:"Number of the rule to rebuild"`
}

func (cmd *RebuildDbCmd) Run(ctx *Context) error {
	config, err := LoadConfig(ctx.ConfigPath.GetAgentConfigPath())
	if err != nil {
		log.Println("Cannot load config", err)
		return err
	}

	if err = ctx.ConfigPath.WriteRCloneConfig(config.RCloneConfig); err != nil {
		log.Println("Cannot write rclone config", err)
		return err
	}

	if cmd.RuleNumber < 0 || cmd.RuleNumber >= len(config.Rules) {
		return fmt.Errorf("Invalid Rule number: %d", cmd.RuleNumber)
	}

	log.Println("Starting DB rebuild process ...")

	rule := config.Rules[cmd.RuleNumber]
	rclone := NewRClone(ctx.ConfigPath)
	orm := NewSQlite(ctx.ConfigPath)
	orm.AddOrUpdateRule(rule.Src, cmd.UUID)

	ruleFolder := ctx.ConfigPath.GetLoopbackFSPath(cmd.UUID)

	filepath.Walk(ruleFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			log.Println("Handling file: ", path)
			entry := orm.CreateEntry(cmd.UUID, path, info.Size())
			if entry.Size == 0 {
				if size, err := rclone.GetSize(entry, rule.Dest); err == nil && size > 0 {
					orm.SendToServer(entry, rule.Dest, size)
				}
			}
		}

		return nil
	})

	log.Println("Finished DB rebuild process ...")

	return nil
}

type TestRuleCmd struct {
	Rule string `arg:"" help:"Name of the rule to test."`
	Path string `arg:"" help:"Path of the file to test." type:"path"`
}

func (cmd *TestRuleCmd) Run(ctx *Context) error {
	// Load config
	config, err := LoadConfig(ctx.ConfigPath.GetAgentConfigPath())
	if err != nil {
		return err
	}

	for _, rule := range config.Rules {
		if string(rule.Type) == cmd.Rule {
			// Get absolute path of the filesystem root handled by the rule
			ruleSrc, err := filepath.Abs(rule.Src)
			if err != nil {
				return err
			}

			// The given file path is not in the rule filesystem
			if !IsSubpath(ruleSrc, cmd.Path, nil) {
				log.Fatal("File is not part of the rule file system")
			}

			printResults := func(path string) {
				if rule.MustBeRemote(path) {
					fmt.Printf("'%s' must be send to remote.\n", path)
				} else {
					fmt.Printf("'%s' must not be send to remote.\n", path)
				}
			}

			if !IsDirectory(cmd.Path) {
				printResults(cmd.Path)
			} else {
				filepath.Walk(cmd.Path, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						printResults(path)
					}
					return nil
				})
			}

			return nil
		}
	}

	log.Fatal("Given rule does not exist.")
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

	orm := NewSQlite(ctx.ConfigPath)

	for _, rule := range config.Rules {
		if _, err := os.Stat(rule.Src); os.IsExist(err) {
			// A folder exists, import it into the loopback folder and in the DB
			importFS(rule, ctx.ConfigPath, orm)
		}
	}

	return SaveConfig(ctx.ConfigPath.GetAgentConfigPath(), config)
}

type CLI struct {
	Debug        bool         `help:"Enable debug mode."`
	ConfigFolder string       `help:"Path to the agent config folder."`
	Sync         SyncCmd      `cmd:"" name:"sync" help:"Run the sync daemon."`
	Rebuild      RebuildDbCmd `cmd:"" name:"rebuild" help:"Rebuild the internal Postgres DB."`
	TestRule     TestRuleCmd  `cmd:"" name:"test-rule" help:"Test a rule on a file."`
	Config       ConfigCmd    `cmd:"" name:"config" help:"Manage the config."`
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
	doSelfUpdate()
	cli := &CLI{
		Debug:        false,
		ConfigFolder: "",
	}
	ctx := kong.Parse(cli)
	err := ctx.Run(&Context{ConfigPath: NewConfigPath(&cli.ConfigFolder, cli.Debug)})
	ctx.FatalIfErrorf(err)
}
