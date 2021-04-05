package main

import (
	"log"
	"os"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
	"github.com/opentracing/opentracing-go"
)

func main() {
	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}

	appLogger := logger.NewApiLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Info("Starting images microservice")
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

	tracer, closer, err := jaeger.InitJaeger(cfg.Jaeger)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}
	appLogger.Info("Jaeger connected")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()
	appLogger.Info("Opentracing connected")

	amqpConn, err := rabbitmq.NewRabbitMQConn(cfg.RabbitMQ)
	if err != nil {
		appLogger.Fatal(err)
	}
	defer amqpConn.Close()

	s3 := images.NewS3Session(cfg)
	appLogger.Infof("AWS S3 Connected : %v", s3.Client.APIVersion)

	server := images.NewServer(appLogger, cfg, tracer, pgxConn, s3)
	appLogger.Fatal(server.Run())
}
