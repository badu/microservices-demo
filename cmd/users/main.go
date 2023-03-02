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
		log.Fatalf("Loading config: %v", err)
	}

	appLogger := logger.NewApiLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Info("Starting user server")
	appLogger.Infof(
		"AppVersion: %s, LogLevel: %s, Mode: %s",
		cfg.GRPCServer.AppVersion,
		cfg.Logger.Level,
		cfg.GRPCServer.Mode,
	)
	appLogger.Infof("Success parsed config: %#v", cfg.GRPCServer.AppVersion)

	pgxConn, err := postgres.NewPgxConn(cfg.Postgres)
	if err != nil {
		appLogger.Fatal("cannot connect to postgres", err)
	}
	defer pgxConn.Close()

	redisClient := redis.NewRedisClient(cfg.Redis)
	appLogger.Info("Redis connected")

	tracer, closer, err := jaeger.InitJaeger(cfg.Jaeger)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}
	appLogger.Info("Jaeger connected")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()
	appLogger.Info("Opentracing connected")

	appLogger.Infof("%-v", pgxConn.Stat())
	appLogger.Infof("%-v", redisClient.PoolStats())

	s := users.NewApplication(&appLogger, cfg, redisClient, pgxConn, tracer)
	appLogger.Fatal(s.Run())
}
