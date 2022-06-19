package main

import "net/url"

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
    value   int
    // can be:
    // * years, months, weeks, days, hours, minutes, seconds    -> time rules
    // * To, Go, Mo, Ko                                         -> size rules
    unit    string
}

type Rule struct {
    // type of rule, must be in the elements above
    Type    RuleType        `json:"type"`

    // paramaters for the rule
    Params interface{}      `json:"params"`

    // source path: can be a file or a folder in the local filesystem
    // if the source is a file, apply the rule
    // if the source is a folder, apply the rule on all its files
    // support shell globbing
    Src     string          `json:"src"`

    // destination path: must be a valid server name
    Dest    string          `json:"dest"`
}

type Config struct {
    Servers map[string]string                   `json:"servers"`            // servers to connect to
    Rules   []Rule                              `json:"rules"`              // rules to apply
    ExcludePatterns []string                    `json:"exclude_patterns"`   // exclude files matching this paterns
    RCloneConfig map[string]map[string]string   `json:"rclone_config"`      // Embedded rclone ini config
}
