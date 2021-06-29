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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func GetCTXSession(c *gin.Context) (*sessions.Session, error) {
	return GetSession().Get(c.Request, viper.GetString("session.cookie"))
}

func SaveCTXSession(c *gin.Context) error {
	return sessions.Save(c.Request, c.Writer)
}

func WithAdmin(f func(c *gin.Context, u *sn.User), re bool) func(c *gin.Context) {
	return WithAdminErr(func(c *gin.Context, u *sn.User) (int, error) {
		f(c, u)
		return 0, nil
	}, re)
}

func WithSignIn(f func(c *gin.Context, u *sn.User), re bool) func(c *gin.Context) {
	return WithSignInErr(func(c *gin.Context, u *sn.User) (int, error) {
		f(c, u)
		return 0, nil
	}, re)
}

func WithAdminErr(f sn.SNAPIFunc, re bool) func(c *gin.Context) {
	return WithSignInErr(func(c *gin.Context, u *sn.User) (int, error) {
		if u.Role == sn.RoleAdmin {
			return f(c, u)
		} else {
			if re {
				c.Redirect(302, "/deny")
			} else {
				c.String(403, "You need admin permission")
			}
		}
		return 0, nil
	}, re)
}

func WithSignInErr(f sn.SNAPIFunc, re bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		res, err := CheckSignIn(c)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}
		if res {
			var u sn.User
			err := GetDB().First(&u, c.MustGet("id")).Error
			if err != nil {
				log.Error(err)
				c.AbortWithStatus(500)
				return
			}
			code, err := f(c, &u)
			if err != nil {
				log.Error(err)
				c.AbortWithStatus(code)
				return
			}
		} else {
			if re {
				c.Redirect(302, "/")
			} else {
				c.String(403, "You need to sign in first")
			}
		}
	}
}

func CheckSignIn(c *gin.Context) (bool, error) {
	if data, err := c.Cookie(viper.GetString("session.cookie")); err == nil && data != "" {
		session, err := GetCTXSession(c)
		if err != nil {
			return false, err
		}

		if session.Values["id"] != nil {
			c.Set("id", session.Values["id"])
			return true, nil
		} else {
			session.Options.MaxAge = -1
			err = SaveCTXSession(c)
			if err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

func FindSessionsByID(id int) (map[string]map[interface{}]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ret := make(map[string]map[interface{}]interface{})
	res, err := GetRedis().Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	pipe := GetRedis().Pipeline()
	piperes := make(map[string]*redis.StringCmd)
	for _, v := range res {
		piperes[v] = pipe.Get(ctx, v)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return nil, err
		}
	}

	for k, v := range piperes {
		var tmp map[interface{}]interface{}
		dec := gob.NewDecoder(strings.NewReader(v.Val()))
		err = dec.Decode(&tmp)
		if err != nil {
			return nil, err
		}
		if tmp["id"] == id {
			ret[k] = tmp
		}
	}
	return ret, nil
}

func DeleteSessionsByID(id int) error {
	s, err := FindSessionsByID(id)
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
		return err
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return err
		}
	}
	return nil
}
