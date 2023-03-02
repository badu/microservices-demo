package grpc_client

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
)

type ClientMiddleware struct {
	logger logger.Logger
	cfg    *config.Config
	tracer opentracing.Tracer
}

func NewClientMiddleware(logger logger.Logger, cfg *config.Config, tracer opentracing.Tracer) *ClientMiddleware {
	return &ClientMiddleware{logger: logger, cfg: cfg, tracer: tracer}
}

func (m *ClientMiddleware) Logger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	meta, _ := metadata.FromIncomingContext(ctx)
	result, err := handler(ctx, req)
	m.logger.Infof("Method: %s, Time: %v, Metadata: %v, Err: %v", info.FullMethod, time.Since(start), meta, err)
	return result, err
}

func (m *ClientMiddleware) GetInterceptor() func(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		m.logger.Infof("call=%v req=%#v reply=%#v time=%v err=%v",
			method, req, reply, time.Since(start), err)
		return err
	}
}

func (m *ClientMiddleware) Tracer() opentracing.Tracer {
	return m.tracer
}
