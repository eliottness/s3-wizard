package main

import (
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

/// The database table for the file entries
type S3NodeTable struct {
	Path            string `gorm:"primaryKey"`
	Size            int64
	Local           bool
	UUID            string
	Server          string
	S3RuleTablePath string
	S3RuleTable     S3RuleTable
}

/// Needed to link the local loopback filesystem
/// with the one we will mount
type S3RuleTable struct {
	UUID string
	Path string `gorm:"primaryKey"`
}

type SQlite struct {
	db     *gorm.DB
	logger *log.Logger
	config *ConfigPath
	batch  []*S3NodeTable
}

func NewSQlite(config *ConfigPath) *SQlite {

	db, err := gorm.Open(sqlite.Open(config.GetDBPath()), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&S3NodeTable{})
	db.AutoMigrate(&S3RuleTable{})
	os.Chmod(config.GetDBPath(), 0600)

	return &SQlite{
		db:     db,
		logger: config.NewLogger("SQLITE: "),
		config: config,
		batch:  make([]*S3NodeTable, 0),
	}
}

func (orm *SQlite) GetNewEntry(rulePath, path string, size int64) *S3NodeTable {
	return &S3NodeTable{
		Path:            path,
		Size:            size,
		Local:           true,
		UUID:            uuid.New().String(),
		Server:          "",
		S3RuleTablePath: rulePath,
	}
}

/// Returns a file entry from the database
func (orm *SQlite) CreateEntry(rulePath, path string, size int64) *S3NodeTable {
	entry := orm.GetNewEntry(rulePath, path, size)
	if result := orm.db.Where("Path = ?", path).Preload("S3RuleTable").FirstOrCreate(entry); result.Error != nil {
		return nil
	}
	return entry
}

/// Returns a file entry from the database
func (orm *SQlite) GetEntry(rulePath, path string, size int64) *S3NodeTable {
	entry := orm.GetNewEntry(rulePath, path, size)
	if result := orm.db.Where("Path = ?", path).Preload("S3RuleTable").First(entry); result.Error != nil {
		return nil
	}
	return entry
}

/// Tell the DB that the file is remote now
func (orm *SQlite) SendToServer(entry *S3NodeTable, server string, size int64) {
	orm.db.Model(entry).Where("Path = ?", entry.Path).Update("Server", server).Update("Local", false).Update("Size", size)
}

func (orm *SQlite) IsEntryLocal(path string) bool {
	var entry []S3NodeTable
	orm.db.Where("Path = ?", path).Limit(1).Find(&entry)
	return len(entry) == 0 || entry[0].Local
}

/// Remove file entry from the database
func (orm *SQlite) DeleteEntry(entry *S3NodeTable) {
	orm.db.Delete(entry.Path)
}

func (orm *SQlite) RenameEntry(oldPath, newPath string) {
	orm.db.Model(&S3NodeTable{}).Where("Path = ?", oldPath).Update("Path", newPath)
}

/// Tell the DB that the file is local now
func (orm *SQlite) RetriveFromServer(entry *S3NodeTable) {
	orm.db.Model(entry).Where("Path = ?", entry.Path).Update("Server", "").Update("Local", true)
}

func (orm *SQlite) GetRule(path string) *S3RuleTable {
	var rule S3RuleTable
	orm.db.Where("Path = ?", path).First(&rule)
	return &rule
}

func (orm *SQlite) AddIfNotExistsRule(path string) *S3RuleTable {
	rule := S3RuleTable{
		Path: path,
		UUID: uuid.New().String(),
	}
	orm.db.Where("Path = ?", path).FirstOrCreate(&rule)
	return &rule
}
