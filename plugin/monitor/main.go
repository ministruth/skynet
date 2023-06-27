package main

import "github.com/MXWXZ/skynet/sn"

type Instance struct{}

func NewPlugin() sn.PluginInstance {
	return new(Instance)
}

func (inst *Instance) PluginLoad() error    { return nil }
func (inst *Instance) PluginEnable() error  { return nil }
func (inst *Instance) PluginDisable() error { return nil }
func (inst *Instance) PluginUnload() error  { return nil }
