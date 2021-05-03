package main

import (
	"errors"
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Plugin config, do NOT change the variable name
var Config = plugins.PluginConfig{
	ID:            uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name:          "monitor",                                              // change to your plugin name
	Dependency:    []plugins.PluginDep{},                                  // if your plugin need dependency, write here
	Version:       "1.0.0",                                                // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1",                                        // skynet version constraints using https://github.com/hashicorp/go-version
}

var defaultField = log.Fields{
	"plugin": Config.ID,
}

var token string

// Delete function below as you wish if you do not need
// All function will be executed after plugin loaded and dependency check

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func PluginInit() error {
	plugins.SPAddSubPath("Service", "Monitor", "/service/"+Config.ID.String(), "", sn.RoleUser, true)
	plugins.SPAddSubPath("Plugin", "Monitor", "/plugin/"+Config.ID.String(), "", sn.RoleAdmin, true)

	plugins.SPAddTemplate("monitor", plugins.SPWithIDPrefix(&Config, "setting"), "templates/setting.tmpl")

	sn.Skynet.PageRouter.GET("/plugin/"+Config.ID.String(), utils.NeedAdmin(PageSetting, true))

	sn.Skynet.APIRouter.GET("/plugin/"+Config.ID.String(), func(c *gin.Context) {
		WSHandler(c.ClientIP(), c.Writer, c.Request)
	})
	sn.Skynet.APIRouter.PATCH("/plugin/"+Config.ID.String(), utils.NeedAdmin(APISaveSetting, false))
	sn.Skynet.APIRouter.PATCH("/plugin/"+Config.ID.String()+"/agent", utils.NeedAdmin(APISaveAgent, false))

	sn.Skynet.Setting.AddSetting(plugins.SPWithIDPrefix(&Config, "token"), "")

	var exist bool
	token, exist = sn.Skynet.Setting.GetSetting(plugins.SPWithIDPrefix(&Config, "token"))
	if !exist {
		log.WithFields(defaultField).Error("Setting token not exist")
		return errors.New("Setting token not exist")
	}

	utils.GetDB().AutoMigrate(&PluginMonitorAgent{})

	var rec []PluginMonitorAgent
	err := utils.GetDB().Find(&rec).Error
	if err != nil {
		return err
	}

	for _, v := range rec {
		agents[int(v.ID)] = &AgentInfo{
			ID:        int(v.ID),
			IP:        v.LastIP,
			Name:      v.Name,
			HostName:  v.Hostname,
			System:    v.System,
			Machine:   v.Machine,
			LastLogin: v.LastLogin,
			Online:    false,
		}
	}
	return nil
}

// PluginEnable will be executed when trigger plugin enabled
func PluginEnable() error {
	return nil
}

// PluginDisable will be executed when trigger plugin disabled, skynet will be reloaded after disabled
func PluginDisable() error {
	return nil
}
