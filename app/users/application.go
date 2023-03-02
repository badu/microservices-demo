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

	"github.com/badu/microservices-demo/app/sessions"
	"github.com/badu/microservices-demo/pkg/grpc_client"

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

type Application struct {
	echo      *echo.Echo
	logger    logger.Logger
	cfg       *config.Config
	redisConn *redis.Client
	pgxPool   *pgxpool.Pool
	tracer    opentracing.Tracer
}

func NewApplication(logger logger.Logger, cfg *config.Config, redisConn *redis.Client, pgxPool *pgxpool.Pool, tracer opentracing.Tracer) *Application {
	return &Application{logger: logger, cfg: cfg, redisConn: redisConn, pgxPool: pgxPool, echo: echo.New(), tracer: tracer}
}

func SessionsGRPCClientFactory(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error) {
	manager := grpc_client.NewClientMiddleware(logger, cfg, tracer)
	return func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error) {
		conn, err := grpc_client.NewGRPCClientServiceConn(ctx, manager, cfg.GRPC.SessionServicePort)
		if err != nil {
			return nil, nil, err
		}
		return conn, sessions.NewAuthorizationServiceClient(conn), nil
	}
}

func (a *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validate := validator.New()
	v1 := a.echo.Group("/api/v1")
	usersGroup := v1.Group("/users")

	userPublisher, err := NewUserPublisher(a.cfg, a.logger)
	if err != nil {
		return errors.Wrap(err, "rabbitmq.NewUserPublisher")
	}

	// queue, err := userPublisher.CreateExchangeAndQueue("images", "resize", "images")

	repository := NewRepository(a.pgxPool)
	redisRepository := NewRedisRepository(a.redisConn, userCachePrefix, userCacheDuration)
	service := NewService(&repository, &redisRepository, a.logger, &userPublisher, SessionsGRPCClientFactory(a.logger, a.cfg, a.tracer))

	mw := NewMiddlewareManager(a.logger, a.cfg, &service)

	httpTransport := NewHTTPServer(usersGroup, &service, a.logger, validate, a.cfg, mw)
	httpTransport.MapUserRoutes()

	consumer := NewConsumer(a.logger, a.cfg, &service)
	if err := consumer.Dial(); err != nil {
		return errors.Wrap(err, "consumer.Dial")
	}
	avatarChan, err := consumer.CreateExchangeAndQueue(UserExchange, AvatarsQueueName, AvatarsBindingKey)
	if err != nil {
		return errors.Wrap(err, "consumer.CreateExchangeAndQueue")
	}
	defer avatarChan.Close()

	consumer.RunConsumers(ctx, cancel)

	a.echo.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok")
	})
	a.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	a.MapRoutes()

	go func() {
		a.logger.Infof("Server is listening on PORT: %s", a.cfg.HttpServer.Port)
		a.echo.Server.ReadTimeout = time.Second * a.cfg.HttpServer.ReadTimeout
		a.echo.Server.WriteTimeout = time.Second * a.cfg.HttpServer.WriteTimeout
		a.echo.Server.MaxHeaderBytes = maxHeaderBytes
		if err := a.echo.StartTLS(a.cfg.HttpServer.Port, certFile, keyFile); err != nil {
			a.logger.Fatalf("Error starting TLS Server: ", err)
		}
	}()

	go func() {
		a.logger.Infof("Starting Debug Server on PORT: %s", a.cfg.HttpServer.PprofPort)
		if err := http.ListenAndServe(a.cfg.HttpServer.PprofPort, http.DefaultServeMux); err != nil {
			a.logger.Errorf("Error PPROF ListenAndServe: %s", err)
		}
	}()

	l, err := net.Listen("tcp", a.cfg.GRPCServer.Port)
	if err != nil {
		return err
	}
	defer l.Close()

	im := grpc_client.NewClientMiddleware(a.logger, a.cfg, a.tracer)

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
			grpcPrometheus.UnaryServerInterceptor,
			grpcRecovery.UnaryServerInterceptor(),
			im.Logger,
		),
	)

	server := NewServer(&service, a.logger)
	RegisterUserServiceServer(grpcServer, &server)
	grpcPrometheus.Register(grpcServer)

	go func() {
		a.logger.Infof("GRPC Server is listening on port: %v", a.cfg.GRPCServer.Port)
		a.logger.Fatal(grpcServer.Serve(l))
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

	a.logger.Info("Server Exited Properly")

	if err := a.echo.Server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "echo.Server.Shutdown")
	}

	grpcServer.GracefulStop()
	a.logger.Info("Server Exited Properly")

	return nil
}

func (a *Application) MapRoutes() {
	a.echo.GET("/swagger/*", echoSwagger.WrapHandler)
	a.echo.Use(middleware.Logger())
	a.echo.Pre(middleware.HTTPSRedirect())
	a.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderXRequestID, csrfTokenHeader},
	}))
	a.echo.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         stackSize,
		DisablePrintStack: true,
		DisableStackAll:   true,
	}))
	a.echo.Use(middleware.RequestID())
	a.echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: gzipLevel,
		Skipper: func(c echo.Context) bool {
			return strings.Contains(c.Request().URL.Path, "swagger")
		},
	}))
	a.echo.Use(middleware.Secure())
	a.echo.Use(middleware.BodyLimit(bodyLimit))
}
