package plugin

import (
	"context"
	"skynet/sn"
	"time"

	"google.golang.org/grpc"

	"skynet/plugin/proto"

	"github.com/hashicorp/go-plugin"
)

const magicKey = "SKYNET_PLUGIN"
const magicValue = "rainhurt"

//go:generate sh -c "protoc -I=./proto --go_out=. --go-grpc_out=. ./proto/*.proto"

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  sn.ProtoVersion,
	MagicCookieKey:   magicKey,
	MagicCookieValue: magicValue,
}

var PluginMap = func(timeout time.Duration) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		"grpc": &PluginGRPCImpl{
			SkynetTimeout: timeout,
		},
	}
}

type PluginError struct {
	Code proto.ErrorCode
}

type PluginHelper interface {
	Eval(str string) (string, error)
}

type PluginAPI interface {
	Enable(h PluginHelper) (*PluginError, error)
	Disable(h PluginHelper) (*PluginError, error)
}

type PluginGRPCImpl struct {
	plugin.NetRPCUnsupportedPlugin
	Impl          PluginAPI
	SkynetTimeout time.Duration
	PluginTimeout time.Duration
}

func (p *PluginGRPCImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServer(s, &GRPCServer{
		impl:    p.Impl,
		broker:  broker,
		timeout: p.PluginTimeout,
	})
	return nil
}

func (p *PluginGRPCImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client:  proto.NewPluginClient(c),
		broker:  broker,
		timeout: p.SkynetTimeout,
	}, nil
}
