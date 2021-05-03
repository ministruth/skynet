package db

import (
	"context"
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

// RedisConfig is config for redis
type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type RedisClient struct {
	redisClient *redis.Client
}

func NewRedis(ctx context.Context, param *RedisConfig) sn.SNDB {
	var ret RedisClient
	ret.redisClient = redis.NewClient(&redis.Options{
		Addr:     param.Address,
		Password: param.Password,
		DB:       param.DB,
	})
	err := ret.redisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatal("Redis connect error: ", err)
	}
	return &ret
}

func (c *RedisClient) GetDB() interface{} {
	if c.redisClient == nil {
		log.Fatal("Redis not init")
	}
	return c.redisClient
}
