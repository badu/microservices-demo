package gateway

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
	"google.golang.org/grpc"

	gatewayComments "github.com/badu/microservices-demo/app/gateway/comments"
	gatewayHotels "github.com/badu/microservices-demo/app/gateway/hotels"
	gatewayUsers "github.com/badu/microservices-demo/app/gateway/users"
	"github.com/badu/microservices-demo/app/sessions"

	"github.com/badu/microservices-demo/app/comments"
	"github.com/badu/microservices-demo/app/hotels"
	"github.com/badu/microservices-demo/app/users"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
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
	tracer    opentracing.Tracer
}

func NewApplication(logger logger.Logger, cfg *config.Config, client *redis.Client, tracer opentracing.Tracer) *Application {
	return &Application{echo: echo.New(), logger: logger, cfg: cfg, redisConn: client, tracer: tracer}
}

func (a *Application) MapRoutes() {
	a.echo.GET("/swagger/*", echoSwagger.WrapHandler)
	cwd, err := os.Getwd()
	if err != nil {
		panic("cannot read working dir")
	}
	cwd = strings.ReplaceAll(cwd, "/cmd", "/app")
	dat, err := os.ReadFile(cwd + "/docs/swagger.json")
	if err != nil {
		panic("cannot read swagger.json?")
	}
	a.echo.GET("/swagger/doc.json", func(c echo.Context) error {
		return c.String(http.StatusOK, string(dat))
	})
	a.echo.Use(middleware.Logger())
	a.echo.Pre(middleware.HTTPSRedirect())
	a.echo.Use(
		middleware.CORSWithConfig(
			middleware.CORSConfig{
				AllowOrigins: []string{"*"},
				AllowHeaders: []string{
					echo.HeaderOrigin,
					echo.HeaderContentType,
					echo.HeaderAccept,
					echo.HeaderXRequestID,
					csrfTokenHeader,
				},
			},
		),
	)
	a.echo.Use(
		middleware.RecoverWithConfig(middleware.RecoverConfig{
			StackSize:         stackSize,
			DisablePrintStack: true,
			DisableStackAll:   true,
		},
		),
	)
	a.echo.Use(middleware.RequestID())
	a.echo.Use(
		middleware.GzipWithConfig(
			middleware.GzipConfig{
				Level: gzipLevel,
				Skipper: func(c echo.Context) bool {
					return strings.Contains(c.Request().URL.Path, "swagger")
				},
			},
		),
	)
	a.echo.Use(middleware.Secure())
	a.echo.Use(middleware.BodyLimit(bodyLimit))

	a.echo.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok")
	})
}

func HotelsGRPCClientFactory(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) func(ctx context.Context) (*grpc.ClientConn, hotels.HotelsServiceClient, error) {
	manager := grpc_client.NewClientMiddleware(logger, cfg, tracer)
	return func(ctx context.Context) (*grpc.ClientConn, hotels.HotelsServiceClient, error) {
		conn, err := grpc_client.NewGRPCClientServiceConn(ctx, manager, cfg.GRPC.HotelsServicePort)
		if err != nil {
			return nil, nil, err
		}
		return conn, hotels.NewHotelsServiceClient(conn), nil
	}
}

func CommentsGRPCClientFactory(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) func(ctx context.Context) (*grpc.ClientConn, comments.CommentsServiceClient, error) {
	manager := grpc_client.NewClientMiddleware(logger, cfg, tracer)
	return func(ctx context.Context) (*grpc.ClientConn, comments.CommentsServiceClient, error) {
		conn, err := grpc_client.NewGRPCClientServiceConn(ctx, manager, cfg.GRPC.CommentsServicePort)
		if err != nil {
			return nil, nil, err
		}
		return conn, comments.NewCommentsServiceClient(conn), nil
	}
}

func UsersGRPCClientFactory(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error) {
	manager := grpc_client.NewClientMiddleware(logger, cfg, tracer)
	return func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error) {
		conn, err := grpc_client.NewGRPCClientServiceConn(ctx, manager, cfg.GRPC.UserServicePort)
		if err != nil {
			return nil, nil, err
		}
		return conn, users.NewUserServiceClient(conn), nil
	}
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

	hotelsRepository := gatewayHotels.NewRepository(a.redisConn)
	hotelsService := gatewayHotels.NewService(a.logger, &hotelsRepository, HotelsGRPCClientFactory(a.logger, a.cfg, a.tracer))

	commentsRepository := gatewayComments.NewRepository(a.redisConn)
	commentsService := gatewayComments.NewService(a.logger, &commentsRepository, CommentsGRPCClientFactory(a.logger, a.cfg, a.tracer))

	usersService := gatewayUsers.NewService(a.logger, SessionsGRPCClientFactory(a.logger, a.cfg, a.tracer), UsersGRPCClientFactory(a.logger, a.cfg, a.tracer))
	userSessionMiddleware := gatewayUsers.NewSessionMiddleware(a.logger, a.cfg, &usersService)

	validate := validator.New()
	// TODO : use error group
	go func() {
		router := echo.New()
		router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
		a.logger.Infof("Metrics server is running on port: %s", a.cfg.Metrics.Port)
		if err := router.Start(a.cfg.Metrics.Port); err != nil {
			a.logger.Error(err)
			cancel()
		}
	}()

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

	v1Routes := a.echo.Group("/api/v1")
	hotelsRoutes := v1Routes.Group("/hotels")
	commentsRoutes := v1Routes.Group("/comments")

	hotelsServer := gatewayHotels.NewServer(a.cfg, hotelsRoutes, a.logger, validate, &hotelsService)
	hotelsServer.MapRoutes(&userSessionMiddleware)

	commentsServer := gatewayComments.NewServer(a.cfg, commentsRoutes, a.logger, validate, &commentsService)
	commentsServer.MapRoutes(&userSessionMiddleware)

	a.MapRoutes()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		a.logger.Errorf("signal.Notify: %v", v)
	case done := <-ctx.Done():
		a.logger.Errorf("ctx.Done: %v", done)
	}

	if err := a.echo.Server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "Server.Shutdown")
	}

	a.logger.Info("Server Exited Properly")
	return nil
}
