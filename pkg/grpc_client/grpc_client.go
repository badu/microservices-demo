package grpc_client

import (
	"context"
	"time"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	traceUtils "github.com/opentracing-contrib/go-grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	backoffLinear = 100 * time.Millisecond
)

func NewGRPCClientServiceConn(ctx context.Context, commonMW *ClientMiddleware, target string) (*grpc.ClientConn, error) {
	opts := []grpcRetry.CallOption{
		grpcRetry.WithBackoff(grpcRetry.BackoffLinear(backoffLinear)),
		grpcRetry.WithCodes(codes.NotFound, codes.Aborted),
	}

	clientGRPCConn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithUnaryInterceptor(traceUtils.OpenTracingClientInterceptor(commonMW.Tracer())),
		grpc.WithUnaryInterceptor(commonMW.GetInterceptor()),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, err
	}

	return clientGRPCConn, nil
}
