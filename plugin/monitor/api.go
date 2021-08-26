package main

import (
	plugins "skynet/plugin"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func APIGetAllAgent(c *gin.Context, u *sn.User) (int, error) {
	var param plugins.PaginationParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}

	count := agentInstance.Len()
	if count > 0 && (param.Page-1)*param.Size < count {
		res := agentInstance.SortValue(func(a, b *shared.AgentElement) bool {
			return a.Key < b.Key
		})
		c.JSON(200, gin.H{"code": 0, "msg": "Get all agent success",
			"data": res[(param.Page-1)*param.Size : utils.IntMin(param.Page*param.Size, len(res))], "total": count})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "Get all agent success", "data": []*shared.AgentInfo{}, "total": count})
	}
	return 0, nil
}

type saveSettingParam struct {
	Token string `json:"token" binding:"required,max=32"`
}

func APISaveSetting(c *gin.Context, u *sn.User) (int, error) {
	var param saveSettingParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": utils.GetIP(c),
	})

	err = sn.Skynet.Setting.Update(tokenKey, param.Token)
	if err != nil {
		return 500, err
	}
	token = param.Token

	agentInstance.Range(func(k int, v *shared.AgentInfo) bool {
		if v.Conn != nil {
			v.Conn.WriteMessage(websocket.CloseMessage, nil)
		}
		return true
	})
	logf.Info("Set token success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set token success"})
	return 0, nil
}

type saveAgentParam struct {
	Name string `json:"name" binding:"required,max=32"`
}

func APISaveAgent(c *gin.Context, u *sn.User) (int, error) {
	var param saveAgentParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 400, err
	}
	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": utils.GetIP(c),
		"id": id,
	})

	if !agentInstance.Has(id) {
		logf.Warn("Agent not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
		return 0, nil
	}

	var rec shared.PluginMonitorAgent
	err = utils.GetDB().First(&rec, id).Error
	if err != nil {
		return 500, err
	}
	rec.Name = param.Name
	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		return 500, err
	}
	agentInstance.MustGet(id).Name = param.Name

	logf.Info("Set name success")
	c.JSON(200, gin.H{"code": 0, "msg": "Set name success"})
	return 0, nil
}

type deleteAgentParam struct {
	ID int `uri:"id" binding:"required,min=1"`
}

func APIDelAgent(c *gin.Context, u *sn.User) (int, error) {
	var param deleteAgentParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		return 400, err
	}
	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": utils.GetIP(c),
		"id": param.ID,
	})

	if !agentInstance.Has(param.ID) {
		logf.Warn("Agent not exist")
		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
		return 0, nil
	}

	pluginAPI.DeleteAllSetting(param.ID)
	err = utils.GetDB().Delete(&shared.PluginMonitorAgent{}, param.ID).Error
	if err != nil {
		return 500, err
	}
	if agentInstance.MustGet(param.ID).Conn != nil {
		agentInstance.MustGet(param.ID).Conn.Close()
	}
	agentInstance.Delete(param.ID)

	logf.Info("Delete agent success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete agent success"})
	return 0, nil
}
