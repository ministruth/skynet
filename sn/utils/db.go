package utils

import (
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	"gorm.io/gorm"
)

func GetDB() *gorm.DB {
	return sn.Skynet.DB.Get().(*gorm.DB)
}

func GetRedis() *redis.Client {
	return sn.Skynet.Redis.Get().(*redis.Client)
}

func GetSession() *redisstore.RedisStore {
	return sn.Skynet.Session.Get().(*redisstore.RedisStore)
}

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
