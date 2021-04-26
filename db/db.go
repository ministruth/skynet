package db

import (
	"context"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBType int

const (
	DBType_Sqlite DBType = iota
)

// DBConfig is config for db
type DBConfig struct {
	Type DBType
	Path string
}

var dbClient *gorm.DB

// InitDB init db
func InitDB(ctx context.Context, param *DBConfig) {
	var err error
	switch param.Type {
	case DBType_Sqlite:
		dbClient, err = gorm.Open(sqlite.Open(param.Path), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			log.Fatal("Failed to connect sqlite database")
		}
		dbClient.AutoMigrate(&Users{})
	}
}

// GetDB get db connection
func GetDB() *gorm.DB {
	if dbClient == nil {
		log.Fatal("DB not init")
	}
	return dbClient
}
