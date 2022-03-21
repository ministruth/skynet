package main

import (
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/tpl"
	"skynet/sn/utils"
	"strings"

	plugins "skynet/plugin"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

func APIGetAllAgent(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Status []shared.AgentStatus `form:"status[]" binding:"dive,min=0,max=2"`
		Text   string               `form:"text"`
		plugins.PaginationParam
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}
	checkStatus := func(s shared.AgentStatus) bool {
		if len(param.Status) == 0 {
			return true
		}
		for _, v := range param.Status {
			if s == v {
				return true
			}
		}
		return false
	}
	checkText := func(s *shared.AgentInfo) bool {
		if param.Text == "" {
			return true
		}
		if strings.Contains(s.ID.String(), param.Text) ||
			strings.Contains(s.Name, param.Text) ||
			strings.Contains(s.IP, param.Text) ||
			strings.Contains(s.Hostname, param.Text) ||
			strings.Contains(s.System, param.Text) ||
			strings.Contains(s.Machine, param.Text) {
			return true
		}
		return false
	}
	var ret tpl.SafeMap[uuid.UUID, *shared.AgentInfo]
	agentInstance.Range(func(k uuid.UUID, v *shared.AgentInfo) bool {
		if checkStatus(v.Status) && checkText(v) {
			ret.Set(k, v)
		}
		return true
	})

	min, max, ok := utils.CalcPage(param.Page, param.Size, ret.Len())
	if !ok {
		Instance.ResponsePage(c, []*shared.AgentInfo{}, ret.Len())
	} else {
		Instance.ResponsePage(c, ret.Values()[min:max], ret.Len())
	}
	return 0, nil
}

func APIGetSetting(c *gin.Context, id uuid.UUID) (int, error) {
	Instance.ResponseData(c, gin.H{"token": token})
	return 0, nil
}

func APIUpdateSetting(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Token string `json:"token" binding:"max=32"`
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := Instance.LogF(c, id, log.Fields{
		"token": param.Token,
	})

	if err := sn.Skynet.Setting.Set(tokenKey, param.Token); err != nil {
		return 500, err
	}
	token = param.Token

	agentInstance.Range(func(k uuid.UUID, v *shared.AgentInfo) bool {
		if v.Conn != nil {
			v.Conn.WriteMessage(websocket.CloseMessage, nil)
		}
		return true
	})
	Instance.LogSuccess(logf, "Set token success")
	Instance.ResponseOK(c)
	return 0, nil
}

// type saveAgentParam struct {
// 	Name string `json:"name" binding:"required,max=32"`
// }

// func APISaveAgent(c *gin.Context, u *sn.User) (int, error) {
// 	var param saveAgentParam
// 	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
// 		return 400, err
// 	}
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil || id <= 0 {
// 		return 400, tracerr.Wrap(err)
// 	}
// 	logf := log.WithFields(defaultField).WithFields(log.Fields{
// 		"ip": utils.GetIP(c),
// 		"id": id,
// 	})

// 	if !agentInstance.Has(id) {
// 		logf.Warn("Agent not exist")
// 		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
// 		return 0, nil
// 	}

// 	var rec shared.PluginMonitorAgent
// 	if err := tracerr.Wrap(utils.GetDB().First(&rec, id).Error); err != nil {
// 		return 500, err
// 	}
// 	rec.Name = param.Name
// 	if err := tracerr.Wrap(utils.GetDB().Save(&rec).Error); err != nil {
// 		return 500, err
// 	}
// 	agentInstance.MustGet(id).Name = param.Name

// 	logf.Info("Set name success")
// 	c.JSON(200, gin.H{"code": 0, "msg": "Set name success"})
// 	return 0, nil
// }

// type deleteAgentParam struct {
// 	ID int `uri:"id" binding:"required,min=1"`
// }

// func APIDelAgent(c *gin.Context, u *sn.User) (int, error) {
// 	var param deleteAgentParam
// 	if err := tracerr.Wrap(c.ShouldBindUri(&param)); err != nil {
// 		return 400, err
// 	}
// 	logf := log.WithFields(defaultField).WithFields(log.Fields{
// 		"ip": utils.GetIP(c),
// 		"id": param.ID,
// 	})

// 	if !agentInstance.Has(param.ID) {
// 		logf.Warn("Agent not exist")
// 		c.JSON(200, gin.H{"code": 1, "msg": "Agent not exist"})
// 		return 0, nil
// 	}

// 	pluginAPI.DeleteAllSetting(param.ID)
// 	err := tracerr.Wrap(utils.GetDB().Delete(&shared.PluginMonitorAgent{}, param.ID).Error)
// 	if err != nil {
// 		return 500, err
// 	}
// 	if agentInstance.MustGet(param.ID).Conn != nil {
// 		agentInstance.MustGet(param.ID).Conn.Close()
// 	}
// 	agentInstance.Delete(param.ID)

// 	logf.Info("Delete agent success")
// 	c.JSON(200, gin.H{"code": 0, "msg": "Delete agent success"})
// 	return 0, nil
// }
