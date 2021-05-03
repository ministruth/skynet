package db

import (
	"context"
	"skynet/sn"

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

type DBClient struct {
	dbClient *gorm.DB
}

func NewDB(ctx context.Context, param *DBConfig) sn.SNDB {
	var ret DBClient
	var err error

	switch param.Type {
	case DBType_Sqlite:
		ret.dbClient, err = gorm.Open(sqlite.Open(param.Path), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			log.Fatal("Failed to connect sqlite database")
		}
		ret.dbClient.AutoMigrate(&sn.Users{}, &sn.Settings{})
	}

	return &ret
}

func (c *DBClient) GetDB() interface{} {
	if c.dbClient == nil {
		log.Fatal("DB not init")
	}
	return c.dbClient
}
