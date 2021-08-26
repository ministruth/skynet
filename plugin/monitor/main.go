package main

import (
	"fmt"
	plugins "skynet/plugin"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var Instance = &plugins.PluginInstance{
	ID:            uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"),
	Name:          "monitor",
	Version:       "1.0.0",
	SkynetVersion: ">= 1.0, < 1.1",
}

type Interface struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &Interface{}
}

var defaultField = log.Fields{
	"plugin": Instance.ID,
}

var (
	token     string
	pluginAPI = NewShared()
	tokenKey  = fmt.Sprintf("plugin_%s_token", Instance.ID.String())
	sharedKey = fmt.Sprintf("plugin_%s", Instance.ID.String())
)

func (p *Interface) Instance() *plugins.PluginInstance {
	return Instance
}

func (p *Interface) PluginInit() error {
	sn.Skynet.Setting.New(tokenKey, "")
	token, _ = sn.Skynet.Setting.Get(tokenKey)
	if token == "" {
		log.WithFields(defaultField).Warn("Token is empty, generate a token for safety")
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

	sn.Skynet.SharedData[sharedKey] = pluginAPI

	Instance.AddStaticRouter(fmt.Sprintf("/css/plugin/%s", Instance.ID), "assets/css")
	Instance.AddStaticRouter(fmt.Sprintf("/js/plugin/%s", Instance.ID), "assets/js")

	Instance.AddSubNav("Service", []*sn.SNNavItem{
		{
			Priority: 16,
			Name:     "Monitor",
			Link:     fmt.Sprintf("/service/%s/monitor", Instance.ID.String()),
			Role:     sn.RoleUser,
		},
		{
			Priority: 17,
			Name:     "Shell",
			Link:     fmt.Sprintf("/service/%s/shell", Instance.ID.String()),
			Role:     sn.RoleAdmin,
		},
	})
	Instance.AddSubNav("Plugin", []*sn.SNNavItem{
		{
			Priority: 16,
			Name:     "Monitor",
			Link:     fmt.Sprintf("/plugin/%s", Instance.ID.String()),
			Role:     sn.RoleAdmin,
		},
	})

	sn.Skynet.Page.AddPage([]*sn.SNPageItem{
		{
			TplName: fmt.Sprintf("plugin_%s_setting", Instance.ID.String()),
			Files:   Instance.WithTplLayerFiles("setting.tmpl"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Monitor",
			Name:    "Monitor",
			Link:    fmt.Sprintf("/plugin/%s", Instance.ID.String()),
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
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				v.Param["_total"] = agentInstance.Len()
				v.Param["token"] = token
				return true
			},
		},
		{
			TplName: fmt.Sprintf("plugin_%s_shell", Instance.ID.String()),
			Files:   Instance.WithTplLayerFiles("shell.tmpl"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Shell",
			Name:    "Shell",
			Link:    fmt.Sprintf("/service/%s/shell", Instance.ID.String()),
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
			TplName: fmt.Sprintf("plugin_%s_monitor", Instance.ID.String()),
			Files:   Instance.WithTplLayerFiles("monitor.tmpl"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Monitor",
			Name:    "Monitor",
			Link:    fmt.Sprintf("/service/%s/monitor", Instance.ID.String()),
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
			Path:   fmt.Sprintf("/plugin/%s/setting", Instance.ID.String()),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APISaveSetting,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/agent", Instance.ID.String()),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetAllAgent,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/agent/:id", Instance.ID.String()),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APISaveAgent,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/agent/:id", Instance.ID.String()),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIDelAgent,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/ws", Instance.ID.String()),
			Method: sn.APIGet,
			Role:   sn.RoleEmpty,
			Func: func(c *gin.Context, u *sn.User) (int, error) {
				WSHandler(utils.GetIP(c), c.Writer, c.Request)
				return 0, nil
			},
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/shell", Instance.ID.String()),
			Method: sn.APIGet,
			Role:   sn.RoleAdmin,
			Func: func(c *gin.Context, u *sn.User) (int, error) {
				ShellHandler(utils.GetIP(c), c.Writer, c.Request)
				return 0, nil
			},
		},
	})
	return nil
}

func (p *Interface) PluginEnable() error {
	return nil
}

func (p *Interface) PluginDisable() error {
	return nil
}

func (p *Interface) PluginFini() error {
	return nil
}
