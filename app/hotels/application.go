package hotels

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcOpenTracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Application struct {
	echo      *echo.Echo
	logger    logger.Logger
	cfg       *config.Config
	redisConn *redis.Client
	pgxPool   *pgxpool.Pool
	tracer    opentracing.Tracer
}

func NewApplication(logger logger.Logger, cfg *config.Config, redisConn *redis.Client, pgxPool *pgxpool.Pool, tracer opentracing.Tracer) Application {
	return Application{logger: logger, cfg: cfg, redisConn: redisConn, pgxPool: pgxPool, echo: echo.New(), tracer: tracer}
}

func (s *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher, err := NewHotelsPublisher(s.cfg, s.logger)
	if err != nil {
		return errors.Wrap(err, "NewHotelsPublisher")
	}

	validate := validator.New()
	repository := NewRepository(s.pgxPool)
	service := NewService(&repository, s.logger, &publisher)

	l, err := net.Listen("tcp", s.cfg.GRPCServer.Port)
	if err != nil {
		return errors.Wrap(err, "net.Listen")
	}
	defer l.Close()

	router := echo.New()
	router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	consumer := NewHotelsConsumer(s.logger, s.cfg, &service)
	if err := consumer.Initialize(); err != nil {
		return errors.Wrap(err, "ConsumerImpl.Initialize")
	}
	consumer.RunConsumers(ctx, cancel)
	defer consumer.CloseChannels()

	go func() {
		if err := router.Start(s.cfg.Metrics.URL); err != nil {
			s.logger.Errorf("router.Start metrics: %v", err)
			cancel()
		}
		s.logger.Infof("Metrics available on: %v", s.cfg.Metrics.URL)
	}()

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
			grpcOpenTracing.UnaryServerInterceptor(),
			grpcPrometheus.UnaryServerInterceptor,
			grpcRecovery.UnaryServerInterceptor(),
		),
	)

	server := NewServer(&service, s.logger, validate)
	RegisterHotelsServiceServer(grpcServer, &server)
	grpcPrometheus.Register(grpcServer)

	go func() {
		s.logger.Infof("GRPC Application is listening on port: %v", s.cfg.GRPCServer.Port)
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

	s.logger.Info("Application Exited Properly")

	if err := s.echo.Server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "echo.Application.Shutdown")
	}

	grpcServer.GracefulStop()
	s.logger.Info("Application Exited Properly")

	return nil
}
