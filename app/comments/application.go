package comments

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcOpenTracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Application struct {
	e       *echo.Echo
	logger  logger.Logger
	cfg     *config.Config
	pgxPool *pgxpool.Pool
	tracer  opentracing.Tracer
}

func NewApplication(logger logger.Logger, cfg *config.Config, pgxPool *pgxpool.Pool, tracer opentracing.Tracer) Application {
	return Application{e: echo.New(), logger: logger, cfg: cfg, pgxPool: pgxPool, tracer: tracer}
}

func UsersGRPCClientFactory(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error) {
	commonMW := grpc_client.NewClientMiddleware(logger, cfg, tracer)
	return func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error) {
		conn, err := grpc_client.NewGRPCClientServiceConn(ctx, commonMW, cfg.GRPC.UserServicePort)
		if err != nil {
			return nil, nil, err
		}
		return conn, users.NewUserServiceClient(conn), nil
	}
}

func (a *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validate := validator.New()

	repository := NewRepository(a.pgxPool)
	service := NewService(&repository, a.logger, UsersGRPCClientFactory(a.logger, a.cfg, a.tracer))
	server := NewServer(&service, a.logger, a.cfg, validate)

	listener, err := net.Listen("tcp", a.cfg.GRPCServer.Port)
	if err != nil {
		return errors.Join(err, errors.New("comments application listener"))
	}
	defer listener.Close()

	go func() {
		router := echo.New()
		router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
		a.logger.Infof("Metrics server is running on port: %s", a.cfg.Metrics.Port)
		if err := router.Start(a.cfg.Metrics.Port); err != nil {
			a.logger.Error(err)
			cancel()
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionIdle: a.cfg.GRPCServer.MaxConnectionIdle * time.Minute,
				Timeout:           a.cfg.GRPCServer.Timeout * time.Second,
				MaxConnectionAge:  a.cfg.GRPCServer.MaxConnectionAge * time.Minute,
				Time:              a.cfg.GRPCServer.Timeout * time.Minute,
			},
		),
		grpc.ChainUnaryInterceptor(
			grpcCtxTags.UnaryServerInterceptor(),
			grpcOpenTracing.UnaryServerInterceptor(),
			grpcPrometheus.UnaryServerInterceptor,
			grpcRecovery.UnaryServerInterceptor(),
		),
	)

	RegisterCommentsServiceServer(grpcServer, &server)

	grpcPrometheus.Register(grpcServer)

	go func() {
		a.logger.Infof("GRPC Server is listening on port: %v", a.cfg.GRPCServer.Port)
		a.logger.Fatal(grpcServer.Serve(listener))
	}()

	if a.cfg.ProductionMode() {
		reflection.Register(grpcServer)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		a.logger.Errorf("signal.Notify: %v", v)
	case done := <-ctx.Done():
		a.logger.Errorf("ctx.Done: %v", done)
	}

	if err := a.e.Server.Shutdown(ctx); err != nil {
		return errors.Join(err, errors.New("comments application server shutdown"))
	}

	grpcServer.GracefulStop()
	a.logger.Info("grpc server shutdown correctly")

	return nil
}
