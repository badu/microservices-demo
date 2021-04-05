package config

import (
	"fmt"
	"time"

	"github.com/badu/microservices-demo/pkg/jaeger"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/postgres"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
	"github.com/badu/microservices-demo/pkg/redis"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	GRPCServer GRPCServer
	HttpServer HttpServer
	Metrics    Metrics
	GRPC       GRPC
	AWS        AWS
	Postgres   postgres.Config
	Redis      redis.Config
	Logger     logger.Config
	Jaeger     jaeger.Config
	RabbitMQ   rabbitmq.Config
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
	AppVersion             string
	Port                   string
	CookieLifeTime         int
	CsrfExpire             int
	SessionID              string
	SessionExpire          int
	Mode                   string
	SessionPrefix          string
	CSRFPrefix             string
	Timeout                time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	MaxConnectionIdle      time.Duration
	MaxConnectionAge       time.Duration
	SessionGrpcServicePort string
}

type HttpServer struct {
	AppVersion        string
	Port              string
	PprofPort         string
	Timeout           time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	CookieLifeTime    int
	SessionCookieName string
	CSRFHeader        string
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
