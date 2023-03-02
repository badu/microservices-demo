package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis config
type Config struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        string
	RedisDefaultDB string
	Password       string
	MinIdleConn    int
	PoolSize       int
	PoolTimeout    int
	DB             int
}

// Returns new redis client
func NewRedisClient(cfg Config) *redis.Client {
	redisHost := cfg.RedisAddr

	if redisHost == "" {
		redisHost = ":6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		MinIdleConns: cfg.MinIdleConn,
		PoolSize:     cfg.PoolSize,
		PoolTimeout:  time.Duration(cfg.PoolTimeout) * time.Second,
		Password:     cfg.Password, // no password set
		DB:           cfg.DB,       // use default DB
	})

	return client
}
