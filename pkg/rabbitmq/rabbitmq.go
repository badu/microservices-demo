package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

// RabbitMQ
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
}

// Initialize new RabbitMQ connection
func NewRabbitMQConn(cfg Config) (*amqp.Connection, error) {
	connAddr := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)
	return amqp.Dial(connAddr)
}
