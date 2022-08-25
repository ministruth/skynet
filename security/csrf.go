package security

import (
	"context"
	"errors"
	"time"

	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

const CSRFHeader = "X-CSRF-Token"

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			token := c.GetHeader(CSRFHeader)
			if token == "" {
				log.New().Debug("Missing CSRF header")
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("csrf.timeout"))*time.Second)
	defer cancel()
	token := utils.RandString(32)
	if err := tracerr.Wrap(db.Redis.SetEX(ctx, viper.GetString("csrf.prefix")+token, "1",
		time.Duration(viper.GetInt("csrf.expire"))*time.Second).Err()); err != nil {
		return "", err
	}
	return token, nil
}

// CheckCSRFToken check csrf token.
func CheckCSRFToken(token string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(viper.GetInt("csrf.timeout"))*time.Second)
	defer cancel()
	ret, err := db.Redis.GetDel(ctx, viper.GetString("csrf.prefix")+token).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	return ret == "1", nil
}
