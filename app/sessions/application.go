package sessions

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/badu/microservices-demo/pkg/grpc_client"

	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	traceUtils "github.com/opentracing-contrib/go-grpc"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Application struct {
	logger    logger.Logger
	cfg       *config.Config
	redisConn *redis.Client
	tracer    opentracing.Tracer
}

func NewApplication(logger logger.Logger, cfg *config.Config, redisConn *redis.Client, tracer opentracing.Tracer) Application {
	return Application{logger: logger, cfg: cfg, redisConn: redisConn, tracer: tracer}
}

func (s *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	im := grpc_client.NewClientMiddleware(s.logger, s.cfg, s.tracer)
	repository := NewRepository(s.redisConn, s.cfg.GRPCServer.SessionPrefix, time.Duration(s.cfg.GRPCServer.SessionExpire)*time.Minute)
	service := NewService(&repository)
	csrfRepository := NewCSRFRepository(s.redisConn, s.cfg.GRPCServer.CSRFPrefix, time.Duration(s.cfg.GRPCServer.CsrfExpire)*time.Minute)
	csrfService := NewCSRFService(&csrfRepository)

	router := echo.New()
	router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	go func() {
		if err := router.Start(s.cfg.Metrics.Port); err != nil {
			s.logger.Errorf("router.Start metrics: %v", err)
			cancel()
		}
		s.logger.Infof("Prometheus Metrics available on: %v", s.cfg.Metrics.Port)
	}()

	l, err := net.Listen("tcp", s.cfg.GRPCServer.Port)
	if err != nil {
		return err
	}
	defer l.Close()

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionIdle: s.cfg.GRPCServer.MaxConnectionIdle * time.Minute,
				Timeout:           s.cfg.GRPCServer.Timeout * time.Second,
				MaxConnectionAge:  s.cfg.GRPCServer.MaxConnectionAge * time.Minute,
				Time:              s.cfg.GRPCServer.Timeout * time.Minute,
			},
		),
		grpc.ChainUnaryInterceptor(
			grpcCtxTags.UnaryServerInterceptor(),
			grpcPrometheus.UnaryServerInterceptor,
			grpcRecovery.UnaryServerInterceptor(),
			traceUtils.OpenTracingServerInterceptor(s.tracer),
			im.Logger,
		),
	)

	server := NewServer(s.logger, &service, &csrfService)
	RegisterAuthorizationServiceServer(grpcServer, &server)
	grpcPrometheus.Register(grpcServer)

	go func() {
		s.logger.Infof("GRPC Server is listening on port: %v", s.cfg.GRPCServer.Port)
		s.logger.Fatal(grpcServer.Serve(l))
	}()

	if s.cfg.ProductionMode() {
		reflection.Register(grpcServer)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		s.logger.Errorf("signal.Notify: %v", v)
	case done := <-ctx.Done():
		s.logger.Errorf("ctx.Done: %v", done)
	}

	if err := router.Shutdown(ctx); err != nil {
		s.logger.Errorf("Metrics router.Shutdown: %v", err)
	}

	grpcServer.GracefulStop()
	s.logger.Info("Server Exited Properly")

	return nil
}
