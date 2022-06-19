package main

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/ini.v1"
)

const CONFIG_FOLDER = ".s3-agent"

type ConfigPath struct {
	folder string
}

func NewConfigPath(user_specified *string) *ConfigPath {
	var configDir string
	if user_specified == nil {
		configDir = getConfigDir()
	} else {
		configDir = *user_specified
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// We don't want anyone to read our config since it will contains credentials
		os.Mkdir(configDir, 0700)
	}

	return &ConfigPath{configDir}
}

func (c *ConfigPath) WriteRCloneConfig(config map[string]map[string]string) error {
	cfg := ini.Empty()
	for section, sectionConfig := range config {
		s := cfg.Section(section)
		for key, value := range sectionConfig {
			s.Key(key).SetValue(value)
		}
	}
	return cfg.SaveTo(c.GetRClonePath())
}

func (c *ConfigPath) GetRClonePath() string {
	// Temporary file, the real unchanging config is in the agent.conf file
	return filepath.Join(c.folder, "rclone.conf.tmp")
}

func (c *ConfigPath) GetAgentPath() string {
	return filepath.Join(c.folder, "agent.conf")
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

func getConfigDir() string {

	var home string
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	} else if runtime.GOOS == "linux" {
		home = os.Getenv("XDG_CONFIG_HOME")
	} else {
		home = os.Getenv("HOME")
	}

	if home == "" {
		panic("No home directory found")
	}

	return filepath.Join(home, CONFIG_FOLDER)
}
