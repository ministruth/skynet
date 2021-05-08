package api

import (
	"errors"
	"io/ioutil"
	"skynet/sn"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vincent-petithory/dataurl"
	"gorm.io/gorm"
)

type userAddParam struct {
	Username string      `form:"username" binding:"required,max=32"`
	Password string      `form:"password" binding:"required"`
	Role     sn.UserRole `form:"role"`
}

func APIAddUser(c *gin.Context, u *sn.Users) (int, error) {
	var param userAddParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip":       c.ClientIP(),
		"id":       u.ID,
		"username": param.Username,
	}

	content, err := ioutil.ReadFile(viper.GetString("default_avatar"))
	if err != nil {
		return 500, err
	}
	_, err = sn.Skynet.User.AddUser(param.Username, param.Password, content, param.Role)
	if err != nil {
		return 500, err
	}
	log.WithFields(fields).Info("Add user success")
	c.JSON(200, gin.H{"code": 0, "msg": "Add user success"})
	return 0, nil
}

type userEditParam struct {
	ID       int32       `form:"id" binding:"required"`
	Username string      `form:"username" binding:"max=32"`
	Password string      `form:"password"`
	Role     sn.UserRole `form:"role"`
	Avatar   string      `form:"avatar"`
}

func APIEditUser(c *gin.Context, u *sn.Users) (int, error) {
	var param userEditParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip":       c.ClientIP(),
		"id":       u.ID,
		"targetID": param.ID,
	}

	if u.ID != param.ID && u.Role < sn.RoleAdmin {
		log.WithFields(fields).Warn("Edit user permission denied")
		c.JSON(200, gin.H{"code": 2, "msg": "Permission denied"})
		return 0, nil
	}
	if u.Role < sn.RoleAdmin {
		param.Role = u.Role // not allow change role
	}

	var avatar []byte
	if param.Avatar != "" {
		tmp, err := dataurl.DecodeString(param.Avatar)
		if err != nil {
			return 500, err
		}
		avatar = tmp.Data
	}
	err = sn.Skynet.User.EditUser(int(param.ID), param.Username, param.Password, param.Role, avatar, u.ID != param.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(fields).Warn("Edit user not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "User not exists"})
		return 0, nil
	} else if err != nil {
		return 500, err
	}
	if param.Username == "" && param.Password == "" && param.Role == sn.RoleEmpty && param.Avatar == "" {
		log.WithFields(fields).Info("Kick user success")
		c.JSON(200, gin.H{"code": 0, "msg": "Kick user success"})
	} else {
		log.WithFields(fields).Info("Edit user success")
		c.JSON(200, gin.H{"code": 0, "msg": "Edit user success"})
	}
	return 0, nil
}

type userDeleteParam struct {
	ID int32 `form:"id" binding:"required"`
}

func APIDeleteUser(c *gin.Context, u *sn.Users) (int, error) {
	var param userDeleteParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip":       c.ClientIP(),
		"id":       u.ID,
		"targetID": param.ID,
	}

	res, err := sn.Skynet.User.DelUser(int(param.ID))
	if err != nil {
		return 500, err
	}

	if !res {
		log.WithFields(fields).Warn("Delete user not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "User not exists"})
		return 0, nil
	}

	log.WithFields(fields).Info("Delete user success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete user success"})
	return 0, nil
}
