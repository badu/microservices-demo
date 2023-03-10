package main

import (
	"log"
	"os"

	"github.com/opentracing/opentracing-go"

	"github.com/badu/microservices-demo/app/gateway"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/redis"
)

func main() {
	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}

	appLogger := logger.NewApiLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Info("Starting API Gateway")
	appLogger.Infof(
		"AppVersion: %s, LogLevel: %s, Mode: %t",
		cfg.HttpServer.AppVersion,
		cfg.Logger.Level,
		cfg.Logger.Development,
	)
	appLogger.Infof("Success parsed config - app version : %#v", cfg.HttpServer.AppVersion)
	appLogger.Infof("HotelsServicePort : %q", cfg.GRPC.HotelsServicePort)

	tracer, closer, err := jaeger.InitJaeger(cfg.Jaeger)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}
	appLogger.Info("connected to Jaeger")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	appLogger.Info("connected to Opentracing")

	redisClient := redis.NewRedisClient(cfg.Redis)
	appLogger.Infof("connected to Redis : %#v", redisClient.PoolStats())

	app := gateway.NewApplication(&appLogger, cfg, redisClient, tracer)
	appLogger.Fatal(app.Run())
}
