package db

import (
	"context"
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	log "github.com/sirupsen/logrus"
)

// SessionConfig is config for session
type SessionConfig struct {
	RedisClient *redis.Client
	Prefix      string
}

type SessionClient struct {
	sessionClient *redisstore.RedisStore
}

func NewSession(ctx context.Context, param *SessionConfig) sn.SNDB {
	var ret SessionClient
	var err error

	ret.sessionClient, err = redisstore.NewRedisStore(ctx, param.RedisClient)
	if err != nil {
		log.Fatal("Redis store error: ", err)
	}
	ret.sessionClient.KeyPrefix(param.Prefix)
	return &ret
}

func (c *SessionClient) Get() interface{} {
	if c.sessionClient == nil {
		log.Fatal("Session not init")
	}
	return c.sessionClient
}
