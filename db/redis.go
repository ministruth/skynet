package db

import (
	"context"
	"skynet/utils/log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

var Redis *redis.Client

// RedisConfig is connection config for redis.
type RedisConfig struct {
	Address  string // redis address
	Password string // redis password
	DB       int    // redis db
}

// NewRedis connect redis with config.
func NewRedis() {
	address := viper.GetString("redis.address")
	password := viper.GetString("redis.password")
	db := viper.GetInt("redis.db")
	timeout := viper.GetInt("redis.timeout")
	log.New().WithFields(log.F{
		"addr": address,
		"db":   db,
	}).Debug("Connecting to redis")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()
	Redis = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})
	err := Redis.Ping(ctx).Err()
	if err != nil {
		log.NewEntry(tracerr.Wrap(err)).Fatal("Failed to connect redis")
	}
	log.New().Debug("Redis connected")
}
