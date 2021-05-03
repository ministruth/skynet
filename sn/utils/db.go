package utils

import (
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	"gorm.io/gorm"
)

func GetDB() *gorm.DB {
	return sn.Skynet.DB.GetDB().(*gorm.DB)
}

func GetRedis() *redis.Client {
	return sn.Skynet.Redis.GetDB().(*redis.Client)
}

func GetSession() *redisstore.RedisStore {
	return sn.Skynet.Session.GetDB().(*redisstore.RedisStore)
}
