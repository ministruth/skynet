package db

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// RedisConfig is config for redis
type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

var redisClient *redis.Client
var sessionClient *redisstore.RedisStore

// InitRedis init redis db
func InitRedis(ctx context.Context, param *RedisConfig) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	redisClient = redis.NewClient(&redis.Options{
		Addr:     param.Address,
		Password: param.Password,
		DB:       param.DB,
	})
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatal("Redis connect error: ", err)
	}
	sessionClient, err = redisstore.NewRedisStore(ctx, redisClient)
	if err != nil {
		log.Fatal("Redis store error: ", err)
	}
	sessionClient.KeyPrefix(viper.GetString("session.prefix"))
}

// GetRedis get redis connection
func GetRedis() *redis.Client {
	if redisClient == nil {
		log.Fatal("Redis not init")
	}
	return redisClient
}

// GetRedis get session store
func GetSession() *redisstore.RedisStore {
	if sessionClient == nil {
		log.Fatal("Session not init")
	}
	return sessionClient
}
