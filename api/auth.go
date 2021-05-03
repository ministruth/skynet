package api

import (
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type authParam struct {
	Username string `form:"username" binding:"required,max=32"`
	Password string `form:"password" binding:"required"`
	Remember bool   `form:"remember"`
}

func APISignIn(c *gin.Context) {
	var param authParam
	err := c.ShouldBind(&param)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}
	fields := log.Fields{
		"ip":       c.ClientIP(),
		"username": param.Username,
	}

	u, res, err := handler.CheckUserPass(param.Username, param.Password)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}

	switch res {
	case 0: // signin
		u.LastLogin = time.Now()
		u.LastIP = c.ClientIP()
		err = utils.GetDB().Save(u).Error
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}

		session, err := utils.GetCTXSession(c)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}
		session.Values["id"] = int(u.ID)
		if param.Remember {
			session.Options.MaxAge = viper.GetInt("session.remember")
		} else {
			session.Options.MaxAge = viper.GetInt("session.expire")
		}
		if err = utils.SaveCTXSession(c); err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}

		log.WithFields(fields).Info("Sign in success")
		c.JSON(200, gin.H{"code": 0, "msg": "Sign in success"})
	default: // invalid
		log.WithFields(fields).Warn("Invalid username or password")
		c.JSON(200, gin.H{"code": 1, "msg": "Invalid username or password"})
	}
}

func APISignOut(c *gin.Context, u *sn.Users) {
	fields := log.Fields{
		"ip": c.ClientIP(),
		"id": u.ID,
	}

	session, err := utils.GetCTXSession(c)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	session.Options.MaxAge = -1
	if err = utils.SaveCTXSession(c); err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	log.WithFields(fields).Info("Sign out success")
	c.JSON(200, gin.H{"code": 0, "msg": "Sign out success"})
}
