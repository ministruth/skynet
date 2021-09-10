package utils

import (
	"context"
	"encoding/gob"
	"skynet/sn"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

// GetCTXSession gets session object from gin context.
func GetCTXSession(c *gin.Context) (*sessions.Session, error) {
	ret, err := GetSession().Get(c.Request, viper.GetString("session.cookie"))
	return ret, tracerr.Wrap(err)
}

// SaveCTXSession saves session object to gin context.
func SaveCTXSession(c *gin.Context) error {
	return tracerr.Wrap(sessions.Save(c.Request, c.Writer))
}

// WithAdmin is middleware for gin handler that need admin privilege.
func WithAdmin(f sn.SNAPIFunc, redirect bool) func(c *gin.Context) {
	return WithSignIn(func(c *gin.Context, u *sn.User) (int, error) {
		if u.Role == sn.RoleAdmin {
			return f(c, u)
		} else {
			if redirect {
				c.Redirect(302, "/deny")
			} else {
				c.String(403, "You need admin permission")
			}
		}
		return 0, nil
	}, redirect)
}

// WithSignIn is middleware for gin handler that need user privilege.
func WithSignIn(f sn.SNAPIFunc, redirect bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		res, err := CheckSignIn(c)
		if err != nil {
			WithTrace(err).Error(err)
			c.AbortWithStatus(500)
			return
		}
		if res {
			var u sn.User
			if err := tracerr.Wrap(GetDB().First(&u, c.MustGet("id")).Error); err != nil {
				WithTrace(err).Error(err)
				c.AbortWithStatus(500)
				return
			}
			code, err := f(c, &u)
			if err != nil {
				WithTrace(err).Error(err)
				c.AbortWithStatus(code)
				return
			}
		} else {
			if redirect {
				c.Redirect(302, "/")
			} else {
				c.String(403, "You need to sign in first")
			}
		}
	}
}

// CheckSignIn checks context signin state and set "id" in the context if state is valid.
func CheckSignIn(c *gin.Context) (bool, error) {
	data, err := c.Cookie(viper.GetString("session.cookie"))
	if err != nil || data == "" {
		return false, tracerr.Wrap(err)
	}
	session, err := GetCTXSession(c)
	if err != nil {
		return false, err
	}
	if session.Values["id"] != nil {
		c.Set("id", session.Values["id"])
		return true, nil
	}
	session.Options.MaxAge = -1
	return false, SaveCTXSession(c)
}

// FindSessionsByID find all sessions associate to user by id.
func FindSessionsByID(userID int) (map[string]map[interface{}]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ret := make(map[string]map[interface{}]interface{})
	res, err := GetRedis().Keys(ctx, viper.GetString("session.prefix")+"*").Result()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	// pipeline to accelerate
	pipe := GetRedis().Pipeline()
	piperes := make(map[string]*redis.StringCmd)
	for _, v := range res {
		piperes[v] = pipe.Get(ctx, v)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return nil, tracerr.Wrap(v.Err())
		}
	}

	// since redis store gob serialized data, we need to decode first
	for k, v := range piperes {
		var tmp map[interface{}]interface{}
		dec := gob.NewDecoder(strings.NewReader(v.Val()))
		if err = tracerr.Wrap(dec.Decode(&tmp)); err != nil {
			return nil, err
		}
		if tmp["id"] == userID { // filter by user id
			ret[k] = tmp
		}
	}
	return ret, nil
}

// DeleteSessionsByID deletes all sessions by id. This equals kick user operation.
func DeleteSessionsByID(userID int) error {
	s, err := FindSessionsByID(userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipe := GetRedis().Pipeline()
	piperes := make(map[string]*redis.IntCmd)
	for k := range s {
		piperes[k] = pipe.Del(ctx, k)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return tracerr.Wrap(v.Err())
		}
	}
	return nil
}
