package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	// Basic rule, when the file is updated, send it to the backend
	// no parameters
	REPLICATE = "REPLICATE"

	// if the source is older than X
	// send it to the backend and leave a dummy behind which will download the source upon opening
	// Parameters example: { value = "3", "unit" = "days" }
	OLDER_THAN = "OLDER_THAN"

	// if the source is newer than X
	// Parameters example: { value = "1", "unit" = "week" }
	NEWER_THAN = "NEWER_THAN"

	// Send the file when larger than X
	// Parameters example: { value = "1", "unit" = "Go" }
	LARGER_THAN = "LARGER_THAN"

	// Send the file when smaller than X
	// Parameters example: { value = "496", "unit" = "Mo" }
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
	Params any `json:"params"`

	// source path: a folder in the local filesystem
	// if the source is a file, apply the rule
	// if the source is a folder, apply the rule on all its files
	// support shell globbing
	Src string `json:"src"`

	// destination path: must be a valid server name
	Dest string `json:"dest"`
}

type Config struct {
	Servers         []string                     `json:"servers"`          // servers to connect to
	Rules           []Rule                       `json:"rules"`            // rules to apply
	ExcludePatterns []string                     `json:"exclude-patterns"` // exclude files matching this paterns
	RCloneConfig    map[string]map[string]string `json:"rclone-config"`    // Embedded rclone ini config
}

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

func SaveConfig(path string, config *Config) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(config)
	if err != nil {
		return err
	}
	return nil
}

func (config *Config) IsValid() error {

	for _, rule := range config.Rules {
		if err := rule.IsValid(); err != nil {
			return err
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

func (rule *Rule) IsValid() error {

	return nil
}
