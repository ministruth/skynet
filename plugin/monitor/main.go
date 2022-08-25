package main

import (
	sp "skynet/plugin"
	"time"

	"github.com/hashicorp/go-plugin"
)

type PluginAPI struct{}

func (m *PluginAPI) Enable(h sp.PluginHelper) (*sp.PluginError, error) {
	return &sp.PluginError{}, nil
}

func (m *PluginAPI) Disable(h sp.PluginHelper) (*sp.PluginError, error) {
	return &sp.PluginError{}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: sp.Handshake,
		Plugins: map[string]plugin.Plugin{
			"grpc": &sp.PluginGRPCImpl{
				Impl:          &PluginAPI{},
				PluginTimeout: time.Second * 10,
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
