package main

import (
	plugins "skynet/plugin"
	"skynet/sn"

	"github.com/google/uuid"
)

// Plugin config, do NOT change the variable name
var Config = plugins.PluginConfig{
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
	},
	Version:       "1.0.0",         // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1", // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      8,               // priority to run PluginInit
}

type PluginInstance struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
	plugins.SPAddSubPath("Plugin", []*sn.SNNavItem{
		{
			Priority: 24,
			Name:     "Shell",
			Link:     "/plugin/" + Config.ID.String(),
			Role:     sn.RoleAdmin,
		},
	})
	sn.Skynet.Page.AddPageItem([]*sn.SNPageItem{
		{
			TplName: plugins.SPWithIDPrefix(&Config, "shell"),
			Files:   plugins.SPWithLayerFiles("shell", "shell"),
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
		},
	})
	// return nil
	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	log.Warn("STARTING!!!")
	// 	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	// 	_, err := t.NewCommand(1, "sleep 10", "aaa", "bbb")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()
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
