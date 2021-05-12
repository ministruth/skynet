package main

import (
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/utils"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type getAgentParam struct {
	Page int `form:"page"`
	Size int `form:"size" binding:"oneof=5 10 20 50"`
}

func APIGetAgent(c *gin.Context, u *sn.Users) (int, error) {
	var param getAgentParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}
	if param.Page <= 0 {
		param.Page = 1
	}

	var sortedAgents AgentSort
	for _, v := range agents {
		sortedAgents = append(sortedAgents, v)
	}
	sort.Stable(sortedAgents)
	low, high := utils.GetSplitPage(len(sortedAgents), param.Page, param.Size)
	if low == -1 {
		c.JSON(200, gin.H{"code": 0, "msg": "Get stat success", "data": sortedAgents})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "Get stat success", "data": sortedAgents[low:high]})
	}
	return 0, nil
}

type saveSettingParam struct {
	Token string `form:"token" binding:"required,max=32"`
}

func APISaveSetting(c *gin.Context, u *sn.Users) (int, error) {
	var param saveSettingParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip": c.ClientIP(),
	}

	err = sn.Skynet.Setting.EditSetting(plugins.SPWithIDPrefix(&Config, "token"), param.Token)
	if err != nil {
		return 500, err
	}
	token = param.Token

	for _, v := range agents {
		if v.Conn != nil {
			v.Conn.WriteMessage(websocket.CloseMessage, nil)
		}
	}
	log.WithFields(defaultField).WithFields(fields).Info("Set token success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set token success"})
	return 0, nil
}

type saveAgentParam struct {
	ID   int    `form:"id" binding:"required"`
	Name string `form:"name" binding:"required,max=32"`
}

func APISaveAgent(c *gin.Context, u *sn.Users) (int, error) {
	var param saveAgentParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip": c.ClientIP(),
		"id": param.ID,
	}

	if _, ok := agents[param.ID]; !ok {
		log.WithFields(defaultField).WithFields(fields).Warn("Agent not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
		return 0, nil
	}

	var rec PluginMonitorAgent
	err = utils.GetDB().First(&rec, param.ID).Error
	if err != nil {
		return 500, err
	}
	rec.Name = param.Name
	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		return 500, err
	}
	agents[param.ID].Name = param.Name

	log.WithFields(defaultField).WithFields(fields).Info("Set name success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set name success"})
	return 0, nil
}

type deleteAgentParam struct {
	ID int `form:"id" binding:"required"`
}

func APIDelAgent(c *gin.Context, u *sn.Users) (int, error) {
	var param deleteAgentParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	fields := log.Fields{
		"ip": c.ClientIP(),
		"id": param.ID,
	}

	if _, ok := agents[param.ID]; !ok {
		log.WithFields(defaultField).WithFields(fields).Warn("Agent not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
		return 0, nil
	}

	err = utils.GetDB().Delete(&PluginMonitorAgent{}, param.ID).Error
	if err != nil {
		return 500, err
	}
	if agents[param.ID].Conn != nil {
		agents[param.ID].Conn.Close()
	}
	delete(agents, param.ID)

	log.WithFields(defaultField).WithFields(fields).Info("Delete agent success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete agent success"})
	return 0, nil
}
