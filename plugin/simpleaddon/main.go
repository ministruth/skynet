package main

import (
	plugins "skynet/plugin"
	"skynet/sn"

	"github.com/google/uuid"
)

// Plugin config, do NOT change the variable name
var Config = &plugins.PluginConfig{
	ID:   uuid.MustParse("7f5282d0-b1b8-4578-8d2a-1949a484aa65"), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name: "simpleaddon",                                          // change to your plugin name
	Dependency: []plugins.PluginDep{
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
	}, // if your plugin need dependency, write here
	Version:       "1.0.0",         // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1", // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      8,               // priority to run PluginInit
}

type PluginInstance struct{}

var pluginAPI = NewShared()

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
	sn.Skynet.SharedData[plugins.SPWithIDPrefix(Config, "")] = pluginAPI
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
