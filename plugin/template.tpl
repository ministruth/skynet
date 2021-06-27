package main

import (
	plugins "skynet/plugin"

	"github.com/google/uuid"
)

// Plugin config, do NOT change the variable name
var Config = plugins.PluginConfig{
	ID:            uuid.MustParse("..."), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name:          "myplugin",            // change to your plugin name
	Dependency:    []plugins.PluginDep{}, // if your plugin need dependency, write here
	Version:       "1.0.0",               // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1",       // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      0,                     // priority to run PluginInit   
}

type PluginInstance struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
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
