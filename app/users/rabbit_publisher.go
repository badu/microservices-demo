package users

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
)

var (
	successPublisherMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rabbitmq_images_success_publish_messages_total",
		Help: "The total number of success RabbitMQ published messages",
	})
	errorPublisherMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rabbitmq_images_error_publish_messages_total",
		Help: "The total number of error RabbitMQ published messages",
	})
)

type PublisherImpl struct {
	amqpConn *amqp.Connection
	cfg      *config.Config
	logger   logger.Logger
}

func NewUserPublisher(cfg *config.Config, logger logger.Logger) (PublisherImpl, error) {
	amqpConn, err := rabbitmq.NewRabbitMQConn(cfg.RabbitMQ)
	if err != nil {
		return PublisherImpl{}, err
	}
	return PublisherImpl{cfg: cfg, logger: logger, amqpConn: amqpConn}, nil
}

func (p *PublisherImpl) CreateExchangeAndQueue(exchange, queueName, bindingKey string) (*amqp.Channel, error) {
	amqpChan, err := p.amqpConn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "p.amqpConn.Channel")
	}

	p.logger.Infof("Declaring exchange: %s", exchange)
	if err := amqpChan.ExchangeDeclare(
		exchange,
		exchangeKind,
		exchangeDurable,
		exchangeAutoDelete,
		exchangeInternal,
		exchangeNoWait,
		nil,
	); err != nil {
		return nil, errors.Wrap(err, "Error ch.ExchangeDeclare")
	}

	queue, err := amqpChan.QueueDeclare(
		queueName,
		queueDurable,
		queueAutoDelete,
		queueExclusive,
		queueNoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error ch.QueueDeclare")
	}

	p.logger.Infof("Declared queue, binding it to exchange: Queue: %v, messageCount: %v, "+
		"consumerCount: %v, exchange: %v, exchange: %v, bindingKey: %v",
		queue.Name,
		queue.Messages,
		queue.Consumers,
		exchange,
		bindingKey,
	)

	err = amqpChan.QueueBind(
		queue.Name,
		bindingKey,
		exchange,
		queueNoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error ch.QueueBind")
	}

	return amqpChan, nil
}

// Publish message
func (p *PublisherImpl) Publish(ctx context.Context, exchange, routingKey, contentType string, headers amqp.Table, body []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "PublisherImpl.Publish")
	defer span.Finish()

	amqpChan, err := p.amqpConn.Channel()
	if err != nil {
		return errors.Wrap(err, "p.amqpConn.Channel")
	}
	defer amqpChan.Close()

	p.logger.Infof("Publishing message Exchange: %s, RoutingKey: %s", exchange, routingKey)

	if err := amqpChan.Publish(
		exchange,
		routingKey,
		publishMandatory,
		publishImmediate,
		amqp.Publishing{
			Headers:      headers,
			ContentType:  contentType,
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.NewV4().String(),
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	); err != nil {
		errorPublisherMessages.Inc()
		return errors.Wrap(err, "ch.Publish")
	}

	successPublisherMessages.Inc()
	return nil
}
