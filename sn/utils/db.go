package utils

import (
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	"gorm.io/gorm"
)

// GetDB return gorm database object, never nil.
func GetDB() *gorm.DB {
	return sn.Skynet.DB.Get().(*gorm.DB)
}

// GetRedis() return go-redis object, never nil.
func GetRedis() *redis.Client {
	return sn.Skynet.Redis.Get().(*redis.Client)
}

// GetSession return redisstore object, never nil.
func GetSession() *redisstore.RedisStore {
	return sn.Skynet.Session.Get().(*redisstore.RedisStore)
}

// DBParseCondition parse sn.SNCondition to gorm.DB object.
//		utils.DBParseCondition(cond).Find(&ret)
func DBParseCondition(cond *sn.SNCondition) *gorm.DB {
	db := GetDB()
	if cond != nil {
		for _, v := range cond.Order {
			db = db.Order(v)
		}
		db = db.Distinct(cond.Distinct...)
		if cond.Limit != nil {
			db = db.Limit(cond.Limit.(int))
		}
		if cond.Offset != nil {
			db = db.Offset(cond.Offset.(int))
		}
		if cond.Where != nil {
			db = db.Where(cond.Where, cond.Args...)
		}
	}
	return db
}
