package db

import (
	"context"
	"skynet/sn"

	"github.com/go-redis/redis/v8"
	"github.com/rbcervilla/redisstore/v8"
	log "github.com/sirupsen/logrus"
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
func NewSession(ctx context.Context, conf *SessionConfig) sn.SNDB {
	var ret sessionClient
	var err error

	ret.sessionClient, err = redisstore.NewRedisStore(ctx, conf.RedisClient)
	if err != nil {
		log.Fatal("Redis store error: ", err)
	}
	ret.sessionClient.KeyPrefix(conf.Prefix)
	return &ret
}

func (c *sessionClient) Get() interface{} {
	if c.sessionClient == nil {
		panic("Session not init")
	}
	return c.sessionClient
}
