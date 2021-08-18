package main

import (
	"errors"
	plugins "skynet/plugin"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Plugin config, do NOT change the variable name
var Config = &plugins.PluginConfig{
	ID:            uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name:          "monitor",                                              // change to your plugin name
	Dependency:    []plugins.PluginDep{},                                  // if your plugin need dependency, write here
	Version:       "1.0.0",                                                // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1",                                        // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      0,                                                      // priority to run PluginInit
}

type PluginInstance struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

var defaultField = log.Fields{
	"plugin": Config.ID,
}

var (
	SettingTokenNotExistError = errors.New("Setting token not exist")
)

var token string

var pluginAPI = NewShared()

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
	sn.Skynet.Setting.New(plugins.SPWithIDPrefix(Config, "token"), "")

	var exist bool
	token, exist = sn.Skynet.Setting.Get(plugins.SPWithIDPrefix(Config, "token"))
	if !exist {
		log.WithFields(defaultField).Error("Setting token not exist")
		return SettingTokenNotExistError
	}

	utils.GetDB().AutoMigrate(&shared.PluginMonitorAgent{}, &shared.PluginMonitorAgentSetting{})

	var rec []shared.PluginMonitorAgent
	err := utils.GetDB().Find(&rec).Error
	if err != nil {
		return err
	}

	for _, v := range rec {
		agentInstance.Set(int(v.ID), &shared.AgentInfo{
			ID:        int(v.ID),
			IP:        v.LastIP,
			Name:      v.Name,
			HostName:  v.Hostname,
			System:    v.System,
			Machine:   v.Machine,
			LastLogin: v.LastLogin,
			Online:    false,
		})
	}

	sn.Skynet.SharedData[plugins.SPWithIDPrefix(Config, "")] = pluginAPI

	plugins.SPAddStatic(Config, "/css"+plugins.SPWithIDPrefixPath(Config, ""), "assets/css")
	plugins.SPAddStatic(Config, "/js"+plugins.SPWithIDPrefixPath(Config, ""), "assets/js")

	plugins.SPAddSubPath("Service", []*sn.SNNavItem{
		{
			Priority: 16,
			Name:     "Monitor",
			Link:     "/service/" + Config.ID.String() + "/monitor",
			Role:     sn.RoleUser,
		},
		{
			Priority: 17,
			Name:     "Shell",
			Link:     "/service/" + Config.ID.String() + "/shell",
			Role:     sn.RoleAdmin,
		},
	})
	plugins.SPAddSubPath("Plugin", []*sn.SNNavItem{
		{
			Priority: 16,
			Name:     "Monitor",
			Link:     "/plugin/" + Config.ID.String(),
			Role:     sn.RoleAdmin,
		},
	})

	sn.Skynet.Page.AddPage([]*sn.SNPageItem{
		{
			TplName: plugins.SPWithIDPrefix(Config, "setting"),
			Files:   plugins.SPWithLayerFiles(Config, "setting"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Monitor",
			Name:    "Monitor",
			Link:    "/plugin/" + Config.ID.String(),
			Role:    sn.RoleAdmin,
			Path: sn.Skynet.Page.GetDefaultPath().WithChild([]*sn.SNPathItem{
				{
					Name: "Plugin",
					Link: "/plugin",
				},
				{
					Name:   "Monitor",
					Active: true,
				},
			}),
			Param: gin.H{
				"token": &token,
			},
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				v.Param["_total"] = agentInstance.Len()
				return true
			},
		},
		{
			TplName: plugins.SPWithIDPrefix(Config, "shell"),
			Files:   plugins.SPWithLayerFiles(Config, "shell"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Shell",
			Name:    "Shell",
			Link:    "/service/" + Config.ID.String() + "/shell",
			Role:    sn.RoleAdmin,
			Path: sn.Skynet.Page.GetDefaultPath().WithChild([]*sn.SNPathItem{
				{
					Name: "Service",
					Link: "#",
				},
				{
					Name:   "Shell",
					Active: true,
				},
			}),
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				v.Param["agents"] = agentInstance.Values()
				return true
			},
		},
		{
			TplName: plugins.SPWithIDPrefix(Config, "monitor"),
			Files:   plugins.SPWithLayerFiles(Config, "monitor"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Monitor",
			Name:    "Monitor",
			Link:    "/service/" + Config.ID.String() + "/monitor",
			Role:    sn.RoleUser,
			Path: sn.Skynet.Page.GetDefaultPath().WithChild([]*sn.SNPathItem{
				{
					Name: "Service",
					Link: "#",
				},
				{
					Name:   "Monitor",
					Active: true,
				},
			}),
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				v.Param["_total"] = agentInstance.Len()
				return true
			},
		},
	})

	sn.Skynet.API.AddAPI([]*sn.SNAPIItem{
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/setting"),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APISaveSetting,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/agent"),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetAllAgent,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/agent/:id"),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APISaveAgent,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/agent/:id"),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIDelAgent,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/ws"),
			Method: sn.APIGet,
			Role:   sn.RoleEmpty,
			Func: func(c *gin.Context, u *sn.User) (int, error) {
				WSHandler(c.ClientIP(), c.Writer, c.Request)
				return 0, nil
			},
		},
		{
			Path:   plugins.SPWithIDPrefixPath(Config, "/shell"),
			Method: sn.APIGet,
			Role:   sn.RoleAdmin,
			Func: func(c *gin.Context, u *sn.User) (int, error) {
				ShellHandler(c.ClientIP(), c.Writer, c.Request)
				return 0, nil
			},
		},
	})
	return nil
}

// PluginEnable will be executed when trigger plugin enabled
func (p *PluginInstance) PluginEnable() error {
	return nil
}

// PluginDisable will be executed when trigger plugin disabled, skynet will be reloaded after disabled
func (p *PluginInstance) PluginDisable() error {
	return nil
}

// PluginFini will be executed after plugin disabled or skynet exit
func (p *PluginInstance) PluginFini() error {
	return nil
}
