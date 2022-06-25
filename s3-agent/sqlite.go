package main

import (
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

/// The database table for the file entries
type S3NodeTable struct {
	Path     string `gorm:"primaryKey"`
	Size     int64
	Local    bool
	UUID     string
	Server   string
	Rulepath string
	Rule     S3RuleTable `gorm:"foreignKey:Rulepath"`
}

/// Needed to link the local loopback filesystem
/// with the one we will mount
type S3RuleTable struct {
	UUID string
	Path string `gorm:"primaryKey"`
}

/// Migrate at start up
func DBSanitize(config *ConfigPath) {
	db := Open(config)
	db.AutoMigrate(&S3NodeTable{})
	db.AutoMigrate(&S3RuleTable{})
	os.Chmod(config.GetDBPath(), 0600)
}

/// Open a connection with the database
func Open(config *ConfigPath) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(config.GetDBPath()), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return db
}

/// Returns a file entry from the database
func GetEntry(db *gorm.DB, path string) *S3NodeTable {
	entry := S3NodeTable{
		Path:   path,
		Size:   0,
		Local:  true,
		UUID:   uuid.New().String(),
		Server: "",
	}
	db.Where("Path = ?", path).FirstOrCreate(&entry)
	return &entry
}

/// Adds a file entry to the database
func NewEntry(rulePath, path string, size int64) *S3NodeTable {
	return &S3NodeTable{
		Path:     path,
		Size:     size,
		Local:    true,
		UUID:     uuid.New().String(),
		Server:   "",
		Rulepath: rulePath,
	}
}

/// Tell the DB that the file is remote now
func SendToServer(db *gorm.DB, entry *S3NodeTable, server string, size int64) {
	db.Model(entry).Where("Path = ?", entry.Path).Update("Server", server).Update("Local", false).Update("Size", size)
}

func IsEntryLocal(db *gorm.DB, path string) bool {
	var entry []S3NodeTable
	db.Where("Path = ?", path).Limit(1).Find(&entry)
	return len(entry) == 0 || entry[0].Local
}

/// Remove file entry from the database
func DeleteEntry(db *gorm.DB, entry *S3NodeTable) {
	db.Delete(entry.Path)
}

func RenameEntry(db *gorm.DB, oldPath, newPath string) {
	db.Model(&S3NodeTable{}).Where("Path = ?", oldPath).Update("Path", newPath)
}

/// Tell the DB that the file is local now
func RetriveFromServer(db *gorm.DB, entry *S3NodeTable) {
	db.Model(entry).Where("Path = ?", entry.Path).Update("Server", "").Update("Local", true)
}

func GetRule(db *gorm.DB, path string) *S3RuleTable {
	var rule S3RuleTable
	db.Where("Path = ?", path).First(&rule)
	return &rule
}

func AddIfNotExistsRule(db *gorm.DB, path string) *S3RuleTable {
	rule := S3RuleTable{
		Path: path,
		UUID: uuid.New().String(),
	}
	db.Where("Path = ?", path).FirstOrCreate(&rule)
	return &rule
}
