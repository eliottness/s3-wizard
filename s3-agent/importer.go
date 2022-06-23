package main

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

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

	return customWalk(rule.Src,
		func(oldPath string, info os.FileInfo) error {
			newPath := filepath.Join(loopbackRoot, oldPath[len(rule.Src):])

			if info.Mode().IsRegular() {
				return importFile(oldPath, newPath, info, rule, db)
			}

			// We need to recreate the symlink correctly
			if info.Mode() & fs.ModeSymlink != 0 {
				pointedOldPath, err := os.Readlink(oldPath)
				if err != nil {
					return err
				}
				pointedNewPath := filepath.Join(loopbackRoot, pointedOldPath[len(rule.Src):])
				return os.Symlink(pointedNewPath, newPath)
			}

			os.Rename(oldPath, newPath)
			return nil
		},
	)
}

func customWalk(dirPath string, walkFunc customWalkFunc) error {

	nodes, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		nodePath := filepath.Join(dirPath, node.Name())
		if node.Mode().IsDir() {
			if err := customWalk(nodePath, walkFunc); err != nil {
				return err
			}
		}

		if err := walkFunc(nodePath, node); err != nil {
			return err
		}
	}
	return nil
}

// Add the file to the DB and send it to remote if we need
func importFile(oldPath, newPath string, info os.FileInfo, rule Rule, db *gorm.DB) error {

	

}
