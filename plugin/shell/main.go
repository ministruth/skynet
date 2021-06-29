package main

import (
	plugins "skynet/plugin"
	monitor "skynet/plugin/monitor/shared"
	simpleaddon "skynet/plugin/simpleaddon/shared"
	"skynet/sn"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Plugin config, do NOT change the variable name
var Config = &plugins.PluginConfig{
	ID:   uuid.MustParse("24a3568a-1147-4f0b-8810-0eac68a7600b"), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name: "shell",                                                // change to your plugin name
	Dependency: []plugins.PluginDep{ // if your plugin need dependency, write here
		{
			ID:      uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"),
			Name:    "monitor",
			Version: ">= 1.0, < 1.1",
		},
		{
			ID:      uuid.MustParse("c1e81895-1f75-4988-9f10-52786b875ec7"),
			Name:    "task",
			Version: ">= 1.0, < 1.1",
		},
		{
			ID:      uuid.MustParse("7f5282d0-b1b8-4578-8d2a-1949a484aa65"),
			Name:    "simpleaddon",
			Version: ">= 1.0, < 1.1",
		},
	},
	Version:       "1.0.0",         // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1", // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      16,              // priority to run PluginInit
}

var defaultField = log.Fields{
	"plugin": Config.ID,
}

type PluginInstance struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
	m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	addon := sn.Skynet.SharedData["plugin_7f5282d0-b1b8-4578-8d2a-1949a484aa65"].(simpleaddon.PluginShared)
	plugins.SPAddSubPath("Plugin", []*sn.SNNavItem{
		{
			Priority: 24,
			Name:     "Shell",
			Link:     "/plugin/" + Config.ID.String(),
			Role:     sn.RoleAdmin,
		},
	})
	addon.WithAddonAPI(Config, InstallTask, UninstallTask, GetVersion)
	sn.Skynet.Page.AddPageItem([]*sn.SNPageItem{
		{
			TplName: plugins.SPWithIDPrefix(Config, "shell"),
			Files:   addon.WithAddonFile(Config, "shell"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Shell",
			Name:    "Shell",
			Link:    "/plugin/" + Config.ID.String(),
			Role:    sn.RoleAdmin,
			Path: sn.Skynet.Page.GetDefaultPath().WithChild([]*sn.SNPathItem{
				{
					Name: "Plugin",
					Link: "/plugin",
				},
				{
					Name:   "Shell",
					Active: true,
				},
			}),
			Param: addon.WithAddonParam(nil, Config, "Shell Agent", "Latest gotty version: "),
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				v.Param["_total"] = len(m.GetAgents())
				return true
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
