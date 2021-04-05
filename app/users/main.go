package users

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
)

const (
	certFile          = "ssl/server.crt"
	keyFile           = "ssl/server.pem"
	maxHeaderBytes    = 1 << 20
	userCachePrefix   = "users:"
	userCacheDuration = time.Minute * 15

	gzipLevel       = 5
	stackSize       = 1 << 10 // 1 KB
	csrfTokenHeader = "X-CSRF-Token"
	bodyLimit       = "2M"
)

type grpcServer struct {
	echo      *echo.Echo
	logger    logger.Logger
	cfg       *config.Config
	redisConn *redis.Client
	pgxPool   *pgxpool.Pool
	tracer    opentracing.Tracer
}

func NewGRPCServer(logger logger.Logger, cfg *config.Config, redisConn *redis.Client, pgxPool *pgxpool.Pool, tracer opentracing.Tracer) *grpcServer {
	return &grpcServer{logger: logger, cfg: cfg, redisConn: redisConn, pgxPool: pgxPool, echo: echo.New(), tracer: tracer}
}

func (s *grpcServer) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validate := validator.New()
	v1 := s.echo.Group("/api/v1")
	usersGroup := v1.Group("/users")

	userPublisher, err := NewUserPublisher(s.cfg, s.logger)
	if err != nil {
		return errors.Wrap(err, "rabbitmq.NewUserPublisher")
	}

	// queue, err := userPublisher.CreateExchangeAndQueue("images", "resize", "images")

	repository := NewRepository(s.pgxPool)
	redisRepository := NewRedisRepository(s.redisConn, userCachePrefix, userCacheDuration)
	service := NewService(repository, redisRepository, s.logger, userPublisher, s.cfg, s.tracer)

	mw := NewMiddlewareManager(s.logger, s.cfg, service)

	uh := NewHTTPServer(usersGroup, service, s.logger, validate, s.cfg, mw)
	uh.MapUserRoutes()

	consumer := NewConsumer(s.logger, s.cfg, service)
	if err := consumer.Dial(); err != nil {
		return errors.Wrap(err, "consumer.Dial")
	}
	avatarChan, err := consumer.CreateExchangeAndQueue(UserExchange, AvatarsQueueName, AvatarsBindingKey)
	if err != nil {
		return errors.Wrap(err, "consumer.CreateExchangeAndQueue")
	}
	defer avatarChan.Close()

	consumer.RunConsumers(ctx, cancel)

	s.echo.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok")
	})
	s.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	s.MapRoutes()

	go func() {
		s.logger.Infof("Server is listening on PORT: %s", s.cfg.HttpServer.Port)
		s.echo.Server.ReadTimeout = time.Second * s.cfg.HttpServer.ReadTimeout
		s.echo.Server.WriteTimeout = time.Second * s.cfg.HttpServer.WriteTimeout
		s.echo.Server.MaxHeaderBytes = maxHeaderBytes
		if err := s.echo.StartTLS(s.cfg.HttpServer.Port, certFile, keyFile); err != nil {
			s.logger.Fatalf("Error starting TLS Server: ", err)
		}
	}()

	go func() {
		s.logger.Infof("Starting Debug Server on PORT: %s", s.cfg.HttpServer.PprofPort)
		if err := http.ListenAndServe(s.cfg.HttpServer.PprofPort, http.DefaultServeMux); err != nil {
			s.logger.Errorf("Error PPROF ListenAndServe: %s", err)
		}
	}()

	l, err := net.Listen("tcp", s.cfg.GRPCServer.Port)
	if err != nil {
		return err
	}
	defer l.Close()

	im := grpc_client.NewClientMiddleware(s.logger, s.cfg, s.tracer)

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
			im.Logger,
		),
	)

	server := NewServer(service, s.logger)
	RegisterUserServiceServer(grpcServer, server)
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

	s.logger.Info("Server Exited Properly")

	if err := s.echo.Server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "echo.Server.Shutdown")
	}

	grpcServer.GracefulStop()
	s.logger.Info("Server Exited Properly")

	return nil
}

func (s *grpcServer) MapRoutes() {
	s.echo.GET("/swagger/*", echoSwagger.WrapHandler)
	s.echo.Use(middleware.Logger())
	s.echo.Pre(middleware.HTTPSRedirect())
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderXRequestID, csrfTokenHeader},
	}))
	s.echo.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         stackSize,
		DisablePrintStack: true,
		DisableStackAll:   true,
	}))
	s.echo.Use(middleware.RequestID())
	s.echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: gzipLevel,
		Skipper: func(c echo.Context) bool {
			return strings.Contains(c.Request().URL.Path, "swagger")
		},
	}))
	s.echo.Use(middleware.Secure())
	s.echo.Use(middleware.BodyLimit(bodyLimit))
}
