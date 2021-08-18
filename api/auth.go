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

func APISignIn(c *gin.Context, u *sn.User) (int, error) {
	var param authParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip":       c.ClientIP(),
		"username": param.Username,
	})

	u, res, err := handler.CheckUserPass(param.Username, param.Password)
	if err != nil {
		return 500, err
	}

	switch res {
	case 0: // signin
		u.LastLogin = time.Now()
		u.LastIP = c.ClientIP()
		err = utils.GetDB().Save(u).Error
		if err != nil {
			return 500, err
		}

		session, err := utils.GetCTXSession(c)
		if err != nil {
			return 500, err
		}
		session.Values["id"] = int(u.ID)
		if param.Remember {
			session.Options.MaxAge = viper.GetInt("session.remember")
		} else {
			session.Options.MaxAge = viper.GetInt("session.expire")
		}
		if err = utils.SaveCTXSession(c); err != nil {
			return 500, err
		}

		logf.Info("Sign in success")
		c.JSON(200, gin.H{"code": 0, "msg": "Sign in success"})
	default: // invalid
		logf.Warn("Invalid username or password")
		c.JSON(200, gin.H{"code": 1, "msg": "Invalid username or password"})
	}
	return 0, nil
}

func APISignOut(c *gin.Context, u *sn.User) (int, error) {
	logf := log.WithFields(log.Fields{
		"ip": c.ClientIP(),
		"id": u.ID,
	})

	session, err := utils.GetCTXSession(c)
	if err != nil {
		return 500, err
	}
	session.Options.MaxAge = -1
	if err = utils.SaveCTXSession(c); err != nil {
		return 500, err
	}
	logf.Info("Sign out success")
	c.JSON(200, gin.H{"code": 0, "msg": "Sign out success"})
	return 0, nil
}

func APIReload(c *gin.Context, u *sn.User) (int, error) {
	c.JSON(200, gin.H{"code": 0, "msg": "Restarting skynet..."})
	go func() {
		time.Sleep(time.Second * 2)
		utils.Restart()
	}()
	return 0, nil
}
