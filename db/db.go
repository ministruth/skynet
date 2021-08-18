package db

import (
	"context"
	"skynet/sn"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBType presents backend database type.
type DBType int

const (
	DBType_Sqlite DBType = iota // sqlite backend
)

// DBConfig is connection config for database.
type DBConfig struct {
	Type DBType // backend database type
	Path string // connect url/file path
}

type dbClient struct {
	dbClient *gorm.DB
}

// NewDB create new database object, exit when facing any error.
func NewDB(ctx context.Context, conf *DBConfig) sn.SNDB {
	var ret dbClient
	var err error

	switch conf.Type {
	case DBType_Sqlite:
		ret.dbClient, err = gorm.Open(sqlite.Open(conf.Path), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent), // disable log
		})
		if err != nil {
			log.Fatal("Failed to connect sqlite database")
		}
		ret.dbClient.AutoMigrate(&sn.User{}, &sn.Setting{}, &sn.Notification{})
	}

	return &ret
}

func (c *dbClient) Get() interface{} {
	if c.dbClient == nil {
		log.Fatal("DB not init")
	}
	return c.dbClient
}
