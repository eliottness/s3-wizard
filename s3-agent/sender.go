package main

import (
	"log"
	"regexp"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"
)

type S3Sender struct {
	rule            *Rule
	fs              *S3FS
	excludePatterns []*regexp.Regexp
    config          *ConfigPath
	stop            chan bool
    logger          *log.Logger
}

func NewS3Sender(rule *Rule, fs *S3FS, excludePattern []string, config *ConfigPath) (*S3Sender, error) {

    s := &S3Sender{
        rule: rule,
        fs: fs,
        excludePatterns: make([]*regexp.Regexp, len(excludePattern)),
        config: config,
        stop: make(chan bool),
        logger: config.NewLogger("SEND: " + rule.Src),
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

func (s *S3Sender) Run(loopTime time.Duration) {
    s.logger.Println("Starting SEND Daemon")
	for {
		select {
		case <-s.stop:
			return
		default:
			s.cycle()
			time.Sleep(loopTime)
		}
	}
}

func (s *S3Sender) Stop() {
    s.logger.Println("Stopping SEND Daemon")
	s.stop <- true
}

func (s *S3Sender) cycle() {

    db := Open(s.config)
    for entry := range s.findConcernedFiles(db) {
        if s.rule.MustBeRemote(entry.Path) {
            if err := s.SendRemote(db, entry); err != nil {
                s.logger.Printf("Could not send to remote %v\n", s.rule.Dest)
            }
        }
    }
}

func (s *S3Sender) SendRemote(db *gorm.DB, entry *S3NodeTable) error {
	// The file does not need to be tracked or the file is already remote
	if entry == nil || !entry.IsLocal {
		return nil
	}

    s.logger.Printf("Sending file: %v -> %v", entry.Path, s.rule.Dest)

	// Lock all file handle related to the file
	s.fs.lockFHs(entry.Path)
	defer s.fs.unlockFHs(entry.Path)

	// TODO Send the file
	// Maybe flock the file but not sure if rclone will work as it will be a child process

    if err := syscall.Truncate(entry.Path, 0); err != nil {
        s.logger.Println("Error truncating the file locally", err)
        return err
    }

	// Replace all file descriptor by the new ones
	if err := s.fs.reloadFds(entry.Path); err != nil {
		s.logger.Println("Error reloading file descriptors", err)
		return err
	}

	SendToServer(db, entry, s.rule.Dest)
    return nil
}

func (s *S3Sender) findConcernedFiles(db *gorm.DB) chan *S3NodeTable {

    var entries []S3NodeTable
    db.Model(&S3NodeTable{}).Find(&entries)

    ch := make(chan *S3NodeTable)

    for _, entry := range entries {

        isNotInTheRule := !strings.HasPrefix(entry.Path, s.rule.Src)
        isPatternExcluded := s.isPatternExcluded(entry.Path)
        isAlreadyRemote := !entry.IsLocal
        if isNotInTheRule || isPatternExcluded || isAlreadyRemote {
            continue
        }

        ch <- &entry
    }

    return ch
}

func (s *S3Sender) isPatternExcluded(path string) bool {
    for _, pattern := range s.excludePatterns {
        if pattern.MatchString(path) {
            return false
        }
    }

    return true
}
