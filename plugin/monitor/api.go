package main

import (
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type saveSettingParam struct {
	Token string `form:"token" binding:"required,max=32"`
}

func APISaveSetting(c *gin.Context, u *sn.Users) {
	var param saveSettingParam
	err := c.ShouldBind(&param)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}
	fields := log.Fields{
		"ip": c.ClientIP(),
	}

	err = sn.Skynet.Setting.EditSetting(plugins.SPWithIDPrefix(&Config, "token"), param.Token)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	token = param.Token

	for _, v := range agents {
		v.Conn.WriteMessage(websocket.CloseMessage, nil)
	}
	log.WithFields(defaultField).WithFields(fields).Info("Set token success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set token success"})
}

type saveAgentParam struct {
	ID   int    `form:"id" binding:"required"`
	Name string `form:"name" binding:"required,max=32"`
}

func APISaveAgent(c *gin.Context, u *sn.Users) {
	var param saveAgentParam
	err := c.ShouldBind(&param)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(400)
		return
	}
	fields := log.Fields{
		"ip": c.ClientIP(),
		"id": param.ID,
	}

	var rec PluginMonitorAgent
	err = utils.GetDB().First(&rec, param.ID).Error
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	rec.Name = param.Name
	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}
	agents[param.ID].Name = param.Name

	log.WithFields(defaultField).WithFields(fields).Info("Set name success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set name success"})
}
