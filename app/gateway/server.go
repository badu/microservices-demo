package gateway

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/badu/microservices-demo/app/gateway/comments"
	"github.com/badu/microservices-demo/app/gateway/hotels"
	"github.com/badu/microservices-demo/app/gateway/users"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/badu/microservices-demo/app/gateway/middlewares"
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

type server struct {
	echo      *echo.Echo
	logger    logger.Logger
	cfg       *config.Config
	redisConn *redis.Client
	tracer    opentracing.Tracer
}

func NewServer(logger logger.Logger, cfg *config.Config, client *redis.Client, tracer opentracing.Tracer) *server {
	return &server{echo: echo.New(), logger: logger, cfg: cfg, redisConn: client, tracer: tracer}
}

func (s *server) MapRoutes() {
	s.echo.GET("/swagger/*", echoSwagger.WrapHandler)
	cwd, err := os.Getwd()
	if err != nil {
		panic("cannot read working dir")
	}
	cwd = strings.ReplaceAll(cwd, "/cmd", "/app")
	dat, err := ioutil.ReadFile(cwd + "/docs/swagger.json")
	if err != nil {
		panic("cannot read swagger.json?")
	}
	s.echo.GET("/swagger/doc.json", func(c echo.Context) error {
		return c.String(http.StatusOK, string(dat))
	})
	s.echo.Use(middleware.Logger())
	s.echo.Pre(middleware.HTTPSRedirect())
	s.echo.Use(
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
	s.echo.Use(
		middleware.RecoverWithConfig(middleware.RecoverConfig{
			StackSize:         stackSize,
			DisablePrintStack: true,
			DisableStackAll:   true,
		},
		),
	)
	s.echo.Use(middleware.RequestID())
	s.echo.Use(
		middleware.GzipWithConfig(
			middleware.GzipConfig{
				Level: gzipLevel,
				Skipper: func(c echo.Context) bool {
					return strings.Contains(c.Request().URL.Path, "swagger")
				},
			},
		),
	)
	s.echo.Use(middleware.Secure())
	s.echo.Use(middleware.BodyLimit(bodyLimit))

	s.echo.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok")
	})
}

func (s *server) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hotelsRepository := hotels.NewRepository(s.redisConn)
	hotelsService := hotels.NewHotelsService(s.logger, s.cfg, hotelsRepository, s.tracer)

	commentsRepository := comments.NewRepository(s.redisConn)

	commentsService := comments.NewService(s.logger, s.cfg, commentsRepository, s.tracer)

	usersService := users.NewService(s.logger, s.cfg, s.tracer)
	mw := middlewares.NewMiddlewareManager(s.logger, s.cfg, usersService)

	validate := validator.New()
	// TODO : use error group
	go func() {
		router := echo.New()
		router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
		s.logger.Infof("Metrics server is running on port: %s", s.cfg.Metrics.Port)
		if err := router.Start(s.cfg.Metrics.Port); err != nil {
			s.logger.Error(err)
			cancel()
		}
	}()

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

	v1Routes := s.echo.Group("/api/v1")
	hotelsRoutes := v1Routes.Group("/hotels")
	commentsRoutes := v1Routes.Group("/comments")

	hotelsServer := hotels.NewServer(s.cfg, hotelsRoutes, s.logger, validate, hotelsService, mw)
	hotelsServer.MapRoutes()

	commentsServer := comments.NewServer(s.cfg, commentsRoutes, s.logger, validate, commentsService, mw)
	commentsServer.MapRoutes()

	s.MapRoutes()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		s.logger.Errorf("signal.Notify: %v", v)
	case done := <-ctx.Done():
		s.logger.Errorf("ctx.Done: %v", done)
	}

	if err := s.echo.Server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "Server.Shutdown")
	}

	s.logger.Info("Server Exited Properly")
	return nil
}
