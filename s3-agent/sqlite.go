package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type S3NodeTable struct {
	Ino     uint64 `gorm:"primaryKey"`
	Size    uint64
	IsLocal bool
	Uuid    string
	Server  string
}

func Open(config *ConfigPath) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(config.GetDBPath()), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&S3NodeTable{})
	return db
}

func GetEntry(db *gorm.DB, ino uint64) *S3NodeTable {
	var entry S3NodeTable
	db.Where("Ino = ?", ino).First(&entry)
	return &entry
}

func NewEntry(ino uint64, size uint64, isLocal bool, uuid string, server string) *S3NodeTable {
	return &S3NodeTable{
		Ino:     ino,
		Size:    size,
		IsLocal: isLocal,
		Uuid:    uuid,
		Server:  server,
	}
}

func SendToServer(db *gorm.DB, entry *S3NodeTable, server string) {
	db.Model(entry).Update("Server", server).Update("IsLocal", false)
}

func DeleteEntry(db *gorm.DB, entry *S3NodeTable) {
	db.Delete(entry, entry.Ino)
}

func RetriveFromServer(db *gorm.DB, entry *S3NodeTable, server string) {
	db.Model(entry).Update("Server", nil).Update("IsLocal", true)
}
