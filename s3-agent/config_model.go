package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/exp/slices"
)

const (
	// if the source is older than X
	// send it to the backend and leave a dummy behind which will download the source upon opening
	// Parameters example: "3min" / "1h45min" (See https://pkg.go.dev/time#Duration)
	OLDER_THAN = "OLDER_THAN"

	// if the source is newer than X
	NEWER_THAN = "NEWER_THAN"

	// Send the file when larger than X
	LARGER_THAN = "LARGER_THAN"

	// Send the file when smaller than X
	SMALLER_THAN = "SMALLER_THAN"

	// User if the file is X
	// Parameters example: "john"
	USER_IS = "USER_IS"
)

type RuleType string

type ValueTypeParamater struct {
	// must be positive
	Value int `json:"value"`
	// can be:
	// * years, months, weeks, days, hours, minutes, seconds    -> time rules
	// * To, Go, Mo, Ko                                         -> size rules
	Unit string `json:"unit"`
}

type Rule struct {
	// type of rule, must be in the elements above
	Type RuleType `json:"type"`

	// paramaters for the rule
	Params string `json:"params"`

	// source path: a folder in the local filesystem
	// if the source is a file, apply the rule
	// if the source is a folder, apply the rule on all its files
	// support regexp
	Src string `json:"src"`

	// destination path: must be a valid server name
	Dest string `json:"dest"`

	// Cron to send the values
	// See Cron format: https://pkg.go.dev/github.com/robfig/cron
	CronSender string `json:"cron-sender"`
}

type Config struct {
	Servers         []string                     `json:"servers"`          // servers to connect to
	Rules           []Rule                       `json:"rules"`            // rules to apply
	ExcludePatterns []string                     `json:"exclude-patterns"` // exclude files matching this paterns
	RCloneConfig    map[string]map[string]string `json:"rclone-config"`    // Embedded rclone ini config
}

// Load configuration from path
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	if err := config.IsValid(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save configuration to path
func SaveConfig(path string, config *Config) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = json.NewEncoder(file).Encode(config); err != nil {
		return err
	}

	os.Chmod(path, 0600)

	return nil
}

// Is Config Valid
func (config *Config) IsValid() error {

	for _, rule := range config.Rules {
		if !slices.Contains(config.Servers, rule.Dest) {
			return fmt.Errorf("Rule with source '%s': No server named '%s'", rule.Src, rule.Dest)
		}
	}

	if len(config.Servers) == 0 {
		return fmt.Errorf("No server specified")
	}

	if len(config.Rules) == 0 {
		return fmt.Errorf("No rule specified")
	}

	if len(config.Rules) > 1 {
		return fmt.Errorf("Only one rule is supported in this version, please upgrade by restarting the agent")
	}

	return nil
}

func (rule *Rule) MustBeRemote(path string) bool {

	switch rule.Type {
	case OLDER_THAN:
		return rule.olderThan(path)
	case NEWER_THAN:
		return rule.newerThan(path)
	default:
		panic(fmt.Errorf("Rule type '%s' not implemented", rule.Type))
	}

}

func (rule *Rule) olderThan(path string) bool {

	fo, err := os.Stat(path)
	if err != nil {
		return false
	}

	paramsDuration, err := time.ParseDuration(rule.Params)
	if err != nil {
		return false
	}

	fileModDuration := time.Now().Sub(fo.ModTime())
	return paramsDuration <= fileModDuration
}

func (rule *Rule) newerThan(path string) bool {

	fo, err := os.Stat(path)
	if err != nil {
		return false
	}

	paramsDuration, err := time.ParseDuration(rule.Params)
	if err != nil {
		return false
	}

	fileModDuration := time.Now().Sub(fo.ModTime())
	return paramsDuration >= fileModDuration
}
