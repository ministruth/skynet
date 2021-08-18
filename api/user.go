package api

import (
	"errors"
	"io/ioutil"
	"skynet/sn"
	"skynet/sn/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vincent-petithory/dataurl"
	"gorm.io/gorm"
)

func APIGetUser(c *gin.Context, u *sn.User) (int, error) {
	var param paginationParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}

	rec, err := sn.Skynet.User.GetAll(&sn.SNCondition{
		Order:  []interface{}{"id " + param.Order},
		Limit:  param.Size,
		Offset: (param.Page - 1) * param.Size,
	})
	if err != nil {
		return 500, err
	}
	count, err := sn.Skynet.User.Count()
	if err != nil {
		return 500, err
	}

	type userInfo struct {
		sn.User
		Online bool
	}
	ret := make([]userInfo, len(rec))
	for i := range rec {
		s, err := utils.FindSessionsByID(int(rec[i].ID))
		if err != nil {
			return 500, err
		}
		copier.Copy(&ret[i].User, rec[i])
		ret[i].Online = len(s) != 0
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Get all user success", "data": ret, "total": count})
	return 0, nil
}

type userAddParam struct {
	Username string      `form:"username" binding:"required,max=32"`
	Password string      `form:"password" binding:"required"`
	Role     sn.UserRole `form:"role"`
}

func APIAddUser(c *gin.Context, u *sn.User) (int, error) {
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
	_, err = sn.Skynet.User.New(param.Username, param.Password, content, param.Role)
	if err != nil {
		return 500, err
	}
	log.WithFields(fields).Info("Add user success")
	c.JSON(200, gin.H{"code": 0, "msg": "Add user success"})
	return 0, nil
}

type userUpdateParam struct {
	Username string      `form:"username" binding:"max=32"`
	Password string      `form:"password"`
	Role     sn.UserRole `form:"role"`
	Avatar   string      `form:"avatar"`
}

func APIUpdateUser(c *gin.Context, u *sn.User) (int, error) {
	var param userUpdateParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip":       c.ClientIP(),
		"id":       u.ID,
		"targetID": id,
	})

	if int(u.ID) != id && u.Role < sn.RoleAdmin {
		logf.Warn("Edit user permission denied")
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
	err = sn.Skynet.User.Update(id, param.Username, param.Password, param.Role, avatar, int(u.ID) != id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logf.Warn("Edit user not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "User not exists"})
		return 0, nil
	} else if err != nil {
		return 500, err
	}
	if param.Username == "" && param.Password == "" && param.Role == sn.RoleEmpty && param.Avatar == "" {
		logf.Info("Kick user success")
		c.JSON(200, gin.H{"code": 0, "msg": "Kick user success"})
	} else {
		logf.Info("Edit user success")
		c.JSON(200, gin.H{"code": 0, "msg": "Edit user success"})
	}
	return 0, nil
}

type userDeleteParam struct {
	ID int32 `uri:"id" binding:"required,min=1"`
}

func APIDeleteUser(c *gin.Context, u *sn.User) (int, error) {
	var param userDeleteParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip":       c.ClientIP(),
		"id":       u.ID,
		"targetID": param.ID,
	})

	res, err := sn.Skynet.User.Delete(int(param.ID))
	if err != nil {
		return 500, err
	}

	if !res {
		logf.Warn("Delete user not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "User not exists"})
		return 0, nil
	}

	logf.Info("Delete user success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete user success"})
	return 0, nil
}
