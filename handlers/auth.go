package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"errors"

	"skynet/db"
	"skynet/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type authParam struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
	Remember int    `form:"remember" binding:"oneof=1 0"`
}

func md5str(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func HashPass(pass string) string {
	return md5str(viper.GetString("database.salt_prefix") + pass + viper.GetString("database.salt_suffix"))
}

// SignIn check token
func SignIn(c *gin.Context) {
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

	u, res, err := CheckUserPass(param.Username, param.Password)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}

	switch res {
	case 0: // signin
		session, err := utils.GetSession(c)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}
		session.Values["id"] = int(u.ID)
		if param.Remember == 1 {
			session.Options.MaxAge = viper.GetInt("session.remember")
		} else {
			session.Options.MaxAge = viper.GetInt("session.expire")
		}
		if err = utils.SaveSession(c); err != nil {
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

func SignOut(c *gin.Context) {
	fields := log.Fields{
		"ip": c.ClientIP(),
		"id": c.MustGet("id"),
	}

	session, err := utils.GetSession(c)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	session.Options.MaxAge = -1
	if err = utils.SaveSession(c); err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	log.WithFields(fields).Info("Sign out success")
	c.JSON(200, gin.H{"code": 0, "msg": "Sign out success"})
}

func CheckUserPass(user string, pass string) (*db.Users, int, error) {
	var rec db.Users
	err := db.GetDB().Where("username = ?", user).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 1, nil
	} else if err != nil {
		return nil, -1, err
	}
	if rec.Password != HashPass(pass) {
		return nil, 2, nil
	}
	return &rec, 0, nil
}
