package api

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type getNotificationParam struct {
	Page int `form:"page" binding:"min=0"`
	Size int `form:"size" binding:"oneof=5 10 20 50"`
}

func APIGetNotification(c *gin.Context, u *sn.Users) (int, error) {
	var param getNotificationParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}
	if param.Page == 0 {
		param.Page = 1
	}

	rec, err := sn.Skynet.Notification.GetAll(&sn.SNCondition{
		Order:  []interface{}{"id desc"},
		Limit:  param.Size,
		Offset: (param.Page - 1) * param.Size,
	})
	if err != nil {
		return 500, err
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Get all notification success", "data": rec})
	return 0, nil
}

func APIDeleteNotification(c *gin.Context, u *sn.Users) (int, error) {
	fields := log.Fields{
		"ip": c.ClientIP(),
	}

	err := sn.Skynet.Notification.DeleteAll()
	if err != nil {
		return 500, err
	}
	log.WithFields(fields).Info("Delete notification")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete notification success"})
	return 0, nil
}
