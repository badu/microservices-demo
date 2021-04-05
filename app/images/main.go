package images

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	traceUtils "github.com/opentracing-contrib/go-grpc"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Server struct {
	logger  logger.Logger
	cfg     *config.Config
	tracer  opentracing.Tracer
	pgxPool *pgxpool.Pool
	s3      *s3.S3
}

func NewServer(logger logger.Logger, cfg *config.Config, tracer opentracing.Tracer, pgxPool *pgxpool.Pool, s3 *s3.S3) *Server {
	return &Server{logger: logger, cfg: cfg, tracer: tracer, pgxPool: pgxPool, s3: s3}
}

func (s *Server) Run() error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	incomingMessages := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rabbitmq_images_incoming_messages_total",
		Help: "The total number of incoming RabbitMQ messages",
	})
	successMessages := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rabbitmq_images_success_messages_total",
		Help: "The total number of success incoming success RabbitMQ messages",
	})
	errorMessages := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rabbitmq_images_error_messages_total",
		Help: "The total number of error incoming success RabbitMQ messages",
	})

	publisher, err := NewPublisher(s.cfg, s.logger, incomingMessages, successMessages, errorMessages)
	if err != nil {
		return errors.Wrap(err, "NewPublisher")
	}
	uploadedChan, err := publisher.CreateExchangeAndQueue(ExchangeName, "uploaded", "uploaded")
	if err != nil {
		return errors.Wrap(err, "publisher.CreateExchangeAndQueue")
	}
	defer uploadedChan.Close()

	repository := NewRepository(s.pgxPool)
	storage := NewAWSStorage(s.cfg, s.s3)
	service := NewService(repository, storage, s.logger, publisher)

	consumer := NewImageConsumer(s.logger, s.cfg, service, incomingMessages, successMessages, errorMessages)
	if err := consumer.Initialize(); err != nil {
		return errors.Wrap(err, "consumer.Initialize")
	}
	consumer.RunConsumers(ctx, cancel)
	defer consumer.CloseChannels()

	l, err := net.Listen("tcp", s.cfg.GRPCServer.Port)
	if err != nil {
		return errors.Wrap(err, "net.Listen")
	}
	defer l.Close()

	router := echo.New()
	router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

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
			grpcPrometheus.UnaryServerInterceptor,
			grpcRecovery.UnaryServerInterceptor(),
			traceUtils.OpenTracingServerInterceptor(s.tracer),
		),
	)

	server := NewImageServer(s.cfg, s.logger, service)
	RegisterImageServiceServer(grpcServer, server)
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

	grpcServer.GracefulStop()
	s.logger.Info("Server Exited Properly")

	return nil
}
