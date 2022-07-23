package main

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

type customWalkFunc func(path string, info fs.FileInfo) error

func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}

// Is Supposed to ask the user if he want to import the current existing folder to
// the s3-agent filesystem
// Must return a rule.Src path empty
func importFS(rule Rule, config *ConfigPath, orm *SQlite) error {

	log.Printf("Content detected at path %v. Starting import process ...", rule.Src)

	if _, err := os.Stat(rule.Src); os.IsNotExist(err) {
		return nil // Nothing to import
	}

	loopbackRoot := config.GetLoopbackFSPath(orm.GetRule(rule.Src).UUID)
	rclone := NewRClone(config)

	log.Println("Import process: Creating folders ...")

	// Creates all folders
	if err := createDirectories(rule.Src, loopbackRoot, rule.Src); err != nil {
		return err
	}

	log.Println("Import process: Copying files ...")

	if err := customWalkFile(rule.Src,
		func(oldPath string, info os.FileInfo) error {
			newPath := filepath.Join(loopbackRoot, oldPath[len(rule.Src)-1:])

			if info.Mode().IsRegular() {
				return importFile(oldPath, newPath, info, rule, orm, rclone)
			}

			// We need to recreate the symlink correctly
			if info.Mode()&fs.ModeSymlink != 0 {
				pointedOldPath, err := os.Readlink(oldPath)
				if err != nil {
					return err
				}
				pointedNewPath := filepath.Join(loopbackRoot, pointedOldPath[len(rule.Src)-1:])
				return os.Symlink(pointedNewPath, newPath)
			}

			// if this rename fails it's ok, it's only some sockets or special files
			moveFile(oldPath, newPath)
			return nil
		},
	); err != nil {
		return err
	}

	log.Println("Import process: Deleting folders ...")

	// We delete all folders. If files are still detected, we let the users handle them.
	if err := deleteDirectories(rule.Src); err != nil {
		return err
	}

	return os.Remove(rule.Src)
}

/// Walk only folders
func createDirectories(dirPath, loopbackRoot, mountPath string) error {

	nodes, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		nodePath := filepath.Join(dirPath, node.Name())
		if !node.Mode().IsDir() {
			continue
		}

		newPath := filepath.Join(loopbackRoot, nodePath[len(mountPath)-1:])
		if err := os.Mkdir(newPath, node.Mode()); err != nil {
			return err
		}

		if err := createDirectories(nodePath, loopbackRoot, mountPath); err != nil {
			return err
		}
	}

	return nil
}

func deleteDirectories(dirPath string) error {

	nodes, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		nodePath := filepath.Join(dirPath, node.Name())
		if !node.Mode().IsDir() {
			continue
		}

		if err := deleteDirectories(nodePath); err != nil {
			return err
		}

		if err := os.Remove(nodePath); err != nil {
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
func importFile(oldPath, newPath string, info os.FileInfo, rule Rule, orm *SQlite, rclone *RClone) error {

	var entries []S3NodeTable
	var entry *S3NodeTable

	orm.db.Model(&entry).Where("Path = ?", oldPath).Find(&entries)

	dest := "local"

	// Update the DB with the new entry
	if len(entries) == 0 {
		entry = orm.CreateEntry(rule.Src, newPath, info.Size())
	} else {
		entry = &entries[0]
		orm.db.Model(&entry).Where("Path = ?", oldPath).Preload("S3RuleTable").Update("Path", entry.Path)
	}

	if rule.MustBeRemote(oldPath) {

		if err := rclone.Send(rule.Dest, oldPath, entry); err != nil {
			return err
		}

		orm.SendToServer(entry, rule.Dest, info.Size())

		if err := syscall.Truncate(oldPath, 0); err != nil {
			log.Println("Error truncating the file locally", err)
			return err
		}

		dest = rule.Dest
	}

	log.Printf("Imported file: %v -> %v", oldPath, dest)
	return os.Rename(oldPath, newPath)
}
