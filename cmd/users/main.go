package main

import (
	"log"
	"os"

	"github.com/opentracing/opentracing-go"

	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
	"github.com/badu/microservices-demo/pkg/redis"
)

func main() {
	configPath := config.GetConfigPath(os.Getenv("config"))

	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("failed loading config: %v", err)
	}

	appLogger := logger.NewApiLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Info("starting user server")
	appLogger.Infof("LogLevel: %s, Mode: %s", cfg.Logger.Level, cfg.GRPCServer.Mode)
	appLogger.Infof("succeeded parsing config (app version: %#v", cfg.GRPCServer.AppVersion)

	pgxConn, err := postgres.NewPgxConn(cfg.Postgres)
	if err != nil {
		appLogger.Fatal("cannot connect to postgres ", err)
	}
	defer pgxConn.Close()

	redisClient := redis.NewRedisClient(cfg.Redis)
	appLogger.Info("connected to Redis")

	tracer, closer, err := jaeger.InitJaeger(cfg.Jaeger)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}
	defer closer.Close()

	appLogger.Info("connected to Jaeger")

	opentracing.SetGlobalTracer(tracer)
	appLogger.Info("connected to Opentracing")

	appLogger.Infof("%#v", pgxConn.Stat())
	appLogger.Infof("%#v", redisClient.PoolStats())

	app := users.NewApplication(&appLogger, cfg, redisClient, pgxConn, tracer)

	appLogger.Fatal(app.Run())
}
