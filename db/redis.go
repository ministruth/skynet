package db

import (
	"context"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/go-redis/redis/v8"
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
func NewRedis(ctx context.Context, conf *RedisConfig) sn.SNDB[*redis.Client] {
	var ret redisClient
	ret.redisClient = redis.NewClient(&redis.Options{
		Addr:     conf.Address,
		Password: conf.Password,
		DB:       conf.DB,
	})
	err := ret.redisClient.Ping(ctx).Err()
	if err != nil {
		utils.WithTrace(err).Fatal("Redis connect error: ", err)
	}
	return &ret
}

func (c *redisClient) Get() *redis.Client {
	if c.redisClient == nil {
		panic("Redis not init")
	}
	return c.redisClient
}
