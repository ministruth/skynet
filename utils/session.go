package utils

import (
	"context"
	"encoding/gob"
	"skynet/db"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func GetSession(c *gin.Context) (*sessions.Session, error) {
	return db.GetSession().Get(c.Request, viper.GetString("session.cookie"))
}

func SaveSession(c *gin.Context) error {
	return sessions.Save(c.Request, c.Writer)
}

func NeedSignIn(f func(c *gin.Context), re bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		res, err := CheckSignIn(c)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}
		if res {
			f(c)
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
		session, err := GetSession(c)
		if err != nil {
			return false, err
		}

		if session.Values["id"] != nil {
			c.Set("id", session.Values["id"])
			return true, nil
		} else {
			session.Options.MaxAge = -1
			err = SaveSession(c)
			if err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

func GetUserFromReq(c *gin.Context) (*db.Users, error) {
	var u db.Users
	err := db.GetDB().First(&u, c.MustGet("id")).Error
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
	}
	return &u, err
}

func FindSessionsByID(id int) (map[string]map[interface{}]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ret := make(map[string]map[interface{}]interface{})
	// pipev:=make()
	res, err := db.GetRedis().Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	pipe := db.GetRedis().Pipeline()
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

	pipe := db.GetRedis().Pipeline()
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
