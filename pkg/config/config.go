package config

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
	"github.com/badu/microservices-demo/pkg/redis"
)

type Config struct {
	HttpServer HttpServer
	Postgres   postgres.Config
	GRPC       GRPC
	RabbitMQ   rabbitmq.Config
	Metrics    Metrics
	Logger     logger.Config
	Redis      redis.Config
	AWS        AWS
	Jaeger     jaeger.Config
	GRPCServer GRPCServer
}

func (c *Config) ProductionMode() bool {
	return c.GRPCServer.Mode == "Production"
}

type AWS struct {
	S3Region         string
	S3EndPoint       string
	S3EndPointMinio  string
	DisableSSL       bool
	S3ForcePathStyle bool
}

type GRPCServer struct {
	SessionPrefix          string
	Port                   string
	SessionGrpcServicePort string
	AppVersion             string
	SessionID              string
	CSRFPrefix             string
	Mode                   string
	CsrfExpire             int
	SessionExpire          int
	Timeout                time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	MaxConnectionIdle      time.Duration
	MaxConnectionAge       time.Duration
	CookieLifeTime         int
}

type HttpServer struct {
	AppVersion        string
	Port              string
	PprofPort         string
	SessionCookieName string
	CSRFHeader        string
	Timeout           time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	CookieLifeTime    int
}

type GRPC struct {
	SessionServicePort  string
	UserServicePort     string
	HotelsServicePort   string
	CommentsServicePort string
	ImagesServicePort   string
}

// Metrics config
type Metrics struct {
	Port        string
	URL         string
	ServiceName string
}

// Load config file from given path
func LoadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigName(filename)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, errors.Wrap(err, "config file not found")
		}
		return nil, err
	}

	return v, nil
}

// Parse config file
func ParseConfig(v *viper.Viper) (*Config, error) {
	c := &Config{}

	err := v.Unmarshal(c)
	if err != nil {
		return nil, err
	}

	fmt.Printf("config : %#v", c)

	if len(c.HttpServer.AppVersion) == 0 {
		return nil, errors.New("app version was not found in config!")
	}

	return c, nil
}

// Get config
func GetConfig(configPath string) (*Config, error) {
	cfgFile, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetConfigPath(configPath string) string {
	switch configPath {
	case "docker":
		return "./config/config-docker"
	case "":
		return "./config/config-local"
	default:
		return configPath
	}
}
