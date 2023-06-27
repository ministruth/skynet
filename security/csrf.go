package security

import (
	"context"
	"errors"
	"time"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

const CSRF_COOKIE = "CSRF_TOKEN"
const CSRF_HEADER = "X-CSRF-Token"

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			token := c.GetHeader(CSRF_HEADER)
			if token == "" {
				log.New().Debug("Missing CSRF header")
				c.AbortWithStatus(400)
				return
			}
			cookieToken, err := c.Cookie(CSRF_COOKIE)
			if err != nil {
				log.New().Debug("Missing CSRF cookie")
				c.AbortWithStatus(400)
				return
			}
			if token != cookieToken {
				log.New().Debug("Mismatch CSRF cookie and header")
				c.AbortWithStatus(400)
				return
			}
			ok, err := CheckCSRFToken(token)
			if err != nil {
				log.NewEntry(err).Error("Failed to check CSRF header")
				c.AbortWithStatus(500)
				return
			}
			if !ok {
				log.New().Debug("Broken CSRF header")
				c.AbortWithStatus(400)
				return
			}
		}
		c.Next()
	}
}

// NewCSRFToken returns new csrf token.
func NewCSRFToken() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("redis.timeout"))*time.Second)
	defer cancel()
	token := utils.RandString(32)
	if err := tracerr.Wrap(sn.Skynet.Redis.SetEx(ctx, viper.GetString("csrf.prefix")+token, "1",
		time.Duration(viper.GetInt("csrf.expire"))*time.Second).Err()); err != nil {
		return "", err
	}
	return token, nil
}

// CheckCSRFToken check csrf token.
func CheckCSRFToken(token string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("redis.timeout"))*time.Second)
	defer cancel()
	ret, err := sn.Skynet.Redis.GetDel(ctx, viper.GetString("csrf.prefix")+token).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	return ret == "1", nil
}
