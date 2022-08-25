package plugin

import (
	"context"
	"time"

	"github.com/MXWXZ/skynet/plugin/proto"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type GRPCClient struct {
	client  proto.PluginClient
	broker  *plugin.GRPCBroker
	timeout time.Duration
}

func (m *GRPCClient) error(e *proto.Error) *PluginError {
	return &PluginError{Code: e.Code}
}

func (m *GRPCClient) initHelper(s **grpc.Server, h PluginHelper) uint32 {
	helperServer := &GRPCPluginHelperServer{impl: h}
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		*s = grpc.NewServer(opts...)
		proto.RegisterPluginHelperServer(*s, helperServer)
		return *s
	}
	brokerID := m.broker.NextId()
	go m.broker.AcceptAndServe(brokerID, serverFunc)
	return brokerID
}

func (m *GRPCClient) Enable(h PluginHelper) (*PluginError, error) {
	var s *grpc.Server
	bid := m.initHelper(&s, h)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	rsp, err := m.client.Enable(ctx, &proto.Empty{Helper: bid})
	if err != nil {
		return nil, err
	}
	s.Stop()
	return m.error(rsp), nil
}

func (m *GRPCClient) Disable(h PluginHelper) (*PluginError, error) {
	var s *grpc.Server
	bid := m.initHelper(&s, h)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	rsp, err := m.client.Disable(ctx, &proto.Empty{Helper: bid})
	if err != nil {
		return nil, err
	}
	s.Stop()
	return m.error(rsp), nil
}

type GRPCServer struct {
	impl    PluginAPI
	broker  *plugin.GRPCBroker
	timeout time.Duration
	proto.UnimplementedPluginServer
}

func (m *GRPCServer) error(e *PluginError) *proto.Error {
	return &proto.Error{Code: e.Code}
}

func (m *GRPCServer) initHelper(id uint32) (*grpc.ClientConn, PluginHelper, error) {
	conn, err := m.broker.Dial(id)
	if err != nil {
		return nil, nil, err
	}
	return conn, &GRPCPluginHelperClient{
		client:  proto.NewPluginHelperClient(conn),
		timeout: m.timeout,
	}, nil
}

func (m *GRPCServer) Enable(
	ctx context.Context,
	req *proto.Empty) (*proto.Error, error) {
	conn, helper, err := m.initHelper(req.Helper)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	v, err := m.impl.Enable(helper)
	return m.error(v), err
}

func (m *GRPCServer) Disable(
	ctx context.Context,
	req *proto.Empty) (*proto.Error, error) {
	conn, helper, err := m.initHelper(req.Helper)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	v, err := m.impl.Disable(helper)
	return m.error(v), err
}

type GRPCPluginHelperClient struct {
	client  proto.PluginHelperClient
	timeout time.Duration
}

func (m *GRPCPluginHelperClient) Eval(str string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	rsp, err := m.client.Eval(ctx, &proto.EvalRequest{
		Str: str,
	})
	if err != nil {
		return "", err
	}
	return rsp.Ret, nil
}

type GRPCPluginHelperServer struct {
	impl PluginHelper
	proto.UnimplementedPluginHelperServer
}

func (m *GRPCPluginHelperServer) Eval(ctx context.Context, req *proto.EvalRequest) (rsp *proto.EvalResponse, err error) {
	rsp = new(proto.EvalResponse)
	rsp.Ret, err = m.impl.Eval(req.Str)
	return
}
