package db

import (
	"context"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
)

// SessionConfig is connection config for session.
type SessionConfig struct {
	RedisClient *redis.Client // redis client for session
	Prefix      string        // session prefix in redis
}

type sessionClient struct {
	sessionClient *redisstore.RedisStore
}

// NewSession create new session object, exit when facing any error.
func NewSession(ctx context.Context, conf *SessionConfig) sn.SNDB[*redisstore.RedisStore] {
	var ret sessionClient
	var err error

	ret.sessionClient, err = redisstore.NewRedisStore(ctx, conf.RedisClient)
	if err != nil {
		utils.WithTrace(err).Fatal("Redis store error: ", err)
	}
	ret.sessionClient.KeyPrefix(conf.Prefix)
	return &ret
}

func (c *sessionClient) Get() *redisstore.RedisStore {
	if c.sessionClient == nil {
		panic("Session not init")
	}
	return c.sessionClient
}
