package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/ini.v1"
)

const configFolder = ".s3-agent"

type ConfigPath struct {
	folder string
	debug  bool
}

func NewConfigPath(userSpecifiedOne *string, debug bool) *ConfigPath {
	var configDir string
	if userSpecifiedOne == nil || *userSpecifiedOne == "" {
		configDir = getConfigDir()
	} else {
		configDir = *userSpecifiedOne
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// We don't want anyone to read our config since it will contains credentials
		if err = os.Mkdir(configDir, 0700); err != nil {
            return nil
        }
	}

	return &ConfigPath{configDir, debug}
}

func (c *ConfigPath) NewLogger(prefix string) *log.Logger {
	return log.New(os.Stderr, prefix, log.Ldate|log.Ltime|log.Lmsgprefix)
}

func (c *ConfigPath) WriteRCloneConfig(config map[string]map[string]string) error {
	cfg := ini.Empty()
	for section, sectionConfig := range config {
		s := cfg.Section(section)
		for key, value := range sectionConfig {
			s.Key(key).SetValue(value)
		}
	}
	if err := cfg.SaveTo(c.GetRClonePath()); err != nil {
		return err
	}

	return os.Chmod(c.GetRClonePath(), 600)
}

func (c *ConfigPath) GetRClonePath() string {
	// Temporary file, the real unchanging config is in the agent.conf file
	return filepath.Join(c.folder, "rclone.conf.tmp")
}

func (c *ConfigPath) GetAgentConfigPath() string {
	return filepath.Join(c.folder, "config.json")
}

func (c *ConfigPath) GetAgentLogPath() string {
	return filepath.Join(c.folder, "agent.log")
}

func (c *ConfigPath) GetAgentPidPath() string {
	return filepath.Join(c.folder, "agent.pid")
}

func (c *ConfigPath) GetDBPath() string {
	return filepath.Join(c.folder, "sqlite.db")
}

func (c *ConfigPath) GetLoopbackFSPath(uuid string) string {
	ruleFolder := filepath.Join(c.folder, uuid)

	if !IsDirectory(ruleFolder) {
		if err := os.Mkdir(ruleFolder, 0700); err != nil {
			panic(err)
		}
	}

	return ruleFolder
}

func getConfigDir() string {

	home := ""
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	}

	if home == "" || runtime.GOOS == "linux" {
		home = os.Getenv("XDG_CONFIG_HOME")
	}

	if home == "" {
		home = os.Getenv("HOME")
	}

	if home == "" {
		panic("No home directory found")
	}

	return filepath.Join(home, configFolder)
}
