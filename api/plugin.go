package api

import (
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type editPluginParam struct {
	ID     string `form:"id" binding:"required,uuid"`
	Enable bool   `form:"enable"`
}

func APIEditPlugin(c *gin.Context, u *sn.Users) {
	var param editPluginParam
	err := c.ShouldBind(&param)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}
	fields := log.Fields{
		"ip":     c.ClientIP(),
		"id":     u.ID,
		"plugin": param.ID,
	}
	id, err := uuid.Parse(param.ID)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}

	if param.Enable {
		err = sn.Skynet.Plugin.EnablePlugin(id)
		if err != nil {
			log.WithFields(fields).Warn("Enable plugin fail")
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return
		}
		log.WithFields(fields).Info("Enable plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Enable plugin success"})
	} else {
		err = sn.Skynet.Plugin.DisablePlugin(id)
		if err != nil {
			log.WithFields(fields).Warn("Disable plugin fail")
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return
		}
		log.WithFields(fields).Info("Disable plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Disable plugin success, reloading"})
		go func() {
			time.Sleep(time.Second * 2)
			utils.Restart()
		}()
	}
}
