package main

import (
	"log"
	"os"

	"github.com/opentracing/opentracing-go"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
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
	appLogger.Info("connected to Jaeger")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()
	appLogger.Info("connected to Opentracing")

	amqpConn, err := rabbitmq.NewRabbitMQConn(cfg.RabbitMQ)
	if err != nil {
		appLogger.Fatal(err)
	}
	defer amqpConn.Close()

	s3 := images.NewS3Session(cfg)
	appLogger.Infof("connected to AWS S3 : %v", s3.Client.APIVersion)

	app := images.NewApplication(&appLogger, cfg, tracer, pgxConn, s3)
	appLogger.Fatal(app.Run())
}
