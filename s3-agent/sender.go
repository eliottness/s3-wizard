package main

import (
	"log"
	"os"
	"regexp"
	"syscall"
)

type S3Sender struct {
	rule            *Rule
	fs              *S3FS
	excludePatterns []*regexp.Regexp
	config          *ConfigPath
	stop            chan bool
	logger          *log.Logger
	orm             *SQlite
}

func NewS3Sender(rule *Rule, fs *S3FS, excludePattern []string, config *ConfigPath, orm *SQlite) (*S3Sender, error) {

	s := &S3Sender{
		rule:            rule,
		fs:              fs,
		excludePatterns: make([]*regexp.Regexp, len(excludePattern)),
		config:          config,
		stop:            make(chan bool),
		logger:          config.NewLogger("SEND: " + rule.Src + " | "),
		orm:             orm,
	}

	// Compile regexps
	for i, pattern := range excludePattern {
		exp, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		s.excludePatterns[i] = exp
	}

	return s, nil
}

func (s *S3Sender) Cycle() {

	s.logger.Println("Running SEND Cycle")

	var entries []S3NodeTable
	s.orm.db.Model(&S3NodeTable{}).Where("Local = ?", true).Preload("S3RuleTable").Find(&entries)

	for _, entry := range entries {
		if entry.S3RuleTablePath != s.rule.Src {
			continue
		}

		if s.isPatternExcluded(entry.Path) {
			continue
		}

		if s.rule.MustBeRemote(entry.Path) {
			if err := s.SendRemote(&entry); err != nil {
				s.logger.Println("Error sending remote:", err)
			}
		}
	}
}

func (s *S3Sender) SendRemote(entry *S3NodeTable) error {

	// The file does not need to be tracked or the file is already remote
	if entry == nil || !entry.Local {
		return nil
	}

	s.logger.Printf("Sending file: %v -> %v", entry.Path, s.rule.Dest)

	// Lock all file handle related to the file
	s.fs.lockFHs(entry.Path)
	defer s.fs.unlockFHs(entry.Path)

	info, err := os.Stat(entry.Path)
	if err != nil {
		return err
	}

	if err := s.fs.rclone.Send(s.rule.Dest, entry.Path, entry); err != nil {
		s.logger.Println("Error sending the file", err)
		return err
	}

	s.orm.SendToServer(entry, s.rule.Dest, info.Size())

	if err := syscall.Truncate(entry.Path, 0); err != nil {
		s.logger.Println("Error truncating the file locally", err)
		return err
	}

	// // Replace all file descriptor by the new ones
	// if err := s.fs.reloadFds(entry.Path); err != nil {
	// 	s.logger.Println("Error reloading file descriptors", err)
	// 	return err
	// }

	return nil
}

func (s *S3Sender) isPatternExcluded(path string) bool {
	for _, pattern := range s.excludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}
