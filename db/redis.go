package db

import (
	"context"
	"time"

	"github.com/MXWXZ/skynet/utils/log"

	"github.com/redis/go-redis/v9"
	"github.com/ztrue/tracerr"
)

// NewRedis connect redis with config.
func NewRedis(dsn string, timeout time.Duration) (*redis.Client, error) {
	log.New().WithFields(log.F{
		"timeout": timeout,
	}).Debug("Connecting to redis")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ret := redis.NewClient(opt)
	if err := tracerr.Wrap(ret.Ping(ctx).Err()); err != nil {
		return nil, err
	}
	log.New().Debug("Redis connected")
	return ret, nil
}
