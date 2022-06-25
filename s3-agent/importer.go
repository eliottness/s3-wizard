package main

import (
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"gorm.io/gorm"
)

type customWalkFunc func(path string, info fs.FileInfo) error

// Is Supposed to ask the user if he want to import the current existing folder to
// the s3-agent filesystem
// Must return a rule.Src path empty
func importFS(rule Rule, config *ConfigPath) error {

	if _, err := os.Stat(rule.Src); os.IsNotExist(err) {
		return nil // Nothing to import
	}

	db := Open(config)
	loopbackRoot := config.GetLoopbackFSPath(GetRule(db, rule.Src).UUID)
	rclone, err := NewRClone(config)
	if err != nil {
		return err
	}

	// Creates all folders
	err = customWalkDir(rule.Src, func(oldPath string, info os.FileInfo) error {
		newPath := filepath.Join(loopbackRoot, oldPath[len(rule.Src):])
		return os.Mkdir(newPath, info.Mode())
	})
	if err != nil {
		return err
	}

	return customWalkFile(rule.Src,
		func(oldPath string, info os.FileInfo) error {
			newPath := filepath.Join(loopbackRoot, oldPath[len(rule.Src):])

			if info.Mode().IsRegular() {
				return importFile(oldPath, newPath, info, rule, db, rclone)
			}

			// We need to recreate the symlink correctly
			if info.Mode()&fs.ModeSymlink != 0 {
				pointedOldPath, err := os.Readlink(oldPath)
				if err != nil {
					return err
				}
				pointedNewPath := filepath.Join(loopbackRoot, pointedOldPath[len(rule.Src):])
				return os.Symlink(pointedNewPath, newPath)
			}

			// if this rename fails it's ok, it's only some sockets or special files
			os.Rename(oldPath, newPath)
			return nil
		},
	)
}

/// Walk only folders
func customWalkDir(dirPath string, walkFunc customWalkFunc) error {

	nodes, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		nodePath := filepath.Join(dirPath, node.Name())
		if !node.Mode().IsDir() {
			continue
		}
		if err := walkFunc(nodePath, node); err != nil {
			return err
		}
		if err := customWalkDir(nodePath, walkFunc); err != nil {
			return err
		}

	}
	return nil
}

func customWalkFile(dirPath string, walkFunc customWalkFunc) error {

	nodes, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		nodePath := filepath.Join(dirPath, node.Name())
		if node.Mode().IsDir() {
			if err := customWalkFile(nodePath, walkFunc); err != nil {
				return err
			}
		} else {
			if err := walkFunc(nodePath, node); err != nil {
				return err
			}
		}
	}
	return nil
}

// Add the file to the DB and send it to remote if we need
func importFile(oldPath, newPath string, info os.FileInfo, rule Rule, db *gorm.DB, rclone *RClone) error {

	var entries []S3NodeTable
	var entry *S3NodeTable
	db.Model(&entry).Where("Path = ?", oldPath).Find(&entries)

	if rule.MustBeRemote(oldPath) {
		entry := GetEntry(db, oldPath)
		rclone.Send(entry)

		if err := syscall.Truncate(entry.Path, 0); err != nil {
			log.Println("Error truncating the file locally", err)
		}
	}

	// Update the DB with the new entry
	if len(entries) == 0 {
		entry = NewEntry(rule.Src, newPath, info.Size())
		db.Model(&entry).Create(entry)
	} else {
		entry = &entries[0]
		db.Model(&entry).Where("Path = ?", oldPath).Update("Path", entry.Path)
	}

	return os.Rename(oldPath, newPath)
}
