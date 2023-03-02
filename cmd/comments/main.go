package main

import (
	"log"
	"os"

	"github.com/badu/microservices-demo/app/comments"

	"github.com/opentracing/opentracing-go"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
)

func main() {
	envConfig := os.Getenv("config")
	configPath := config.GetConfigPath(envConfig)
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}

	appLogger := logger.NewApiLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Info("Starting Comments service")
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

	appLogger.Infof("%-v", pgxConn.Stat())

	tracer, closer, err := jaeger.InitJaeger(cfg.Jaeger)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}

	appLogger.Info("Jaeger connected")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	appLogger.Info("Opentracing connected")

	app := comments.NewApplication(&appLogger, cfg, pgxConn, tracer)

	appLogger.Fatal(app.Run())
}
