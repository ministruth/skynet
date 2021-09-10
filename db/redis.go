package db

import (
	"context"
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

// RedisConfig is connection config for redis.
type RedisConfig struct {
	Address  string // redis address
	Password string // redis password
	DB       int    // redis db
}

type redisClient struct {
	redisClient *redis.Client
}

// NewRedis create new redis object, exit when facing any error.
func NewRedis(ctx context.Context, conf *RedisConfig) sn.SNDB {
	var ret redisClient
	ret.redisClient = redis.NewClient(&redis.Options{
		Addr:     conf.Address,
		Password: conf.Password,
		DB:       conf.DB,
	})
	err := ret.redisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatal("Redis connect error: ", err)
	}
	return &ret
}

func (c *redisClient) Get() interface{} {
	if c.redisClient == nil {
		panic("Redis not init")
	}
	return c.redisClient
}
