package impl

import (
	"context"
	"errors"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

// NewCSRFToken returns new csrf token.
func NewCSRFToken() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("csrf.timeout"))*time.Second)
	defer cancel()
	token := utils.RandString(32)
	if err := tracerr.Wrap(sn.Skynet.GetRedis().SetEX(ctx, viper.GetString("csrf.prefix")+token, "1",
		time.Duration(viper.GetInt("csrf.expire"))*time.Second).Err()); err != nil {
		return "", err
	}
	return token, nil
}

// CheckCSRFToken check csrf token.
func CheckCSRFToken(token string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("csrf.timeout"))*time.Second)
	defer cancel()
	ret, err := sn.Skynet.GetRedis().GetDel(ctx, viper.GetString("csrf.prefix")+token).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	return ret == "1", nil
}
