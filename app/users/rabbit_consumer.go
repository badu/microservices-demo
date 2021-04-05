package users

import (
	"context"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
)

type consumerImpl struct {
	amqpConn         *amqp.Connection
	logger           logger.Logger
	cfg              *config.Config
	service          Service
	incomingMessages prometheus.Counter
	successMessages  prometheus.Counter
	errorMessages    prometheus.Counter
}

func NewConsumer(logger logger.Logger, cfg *config.Config, service Service) *consumerImpl {
	return &consumerImpl{
		logger:  logger,
		cfg:     cfg,
		service: service,
		incomingMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rabbitmq_images_incoming_messages_total",
			Help: "The total number of incoming RabbitMQ messages",
		}),
		successMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rabbitmq_images_success_messages_total",
			Help: "The total number of success incoming success RabbitMQ messages",
		}),
		errorMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rabbitmq_images_error_messages_total",
			Help: "The total number of error incoming success RabbitMQ messages",
		}),
	}
}

func (c *consumerImpl) Dial() error {
	conn, err := rabbitmq.NewRabbitMQConn(c.cfg.RabbitMQ)
	if err != nil {
		return err
	}
	c.amqpConn = conn
	return nil
}

// Consume messages
func (c *consumerImpl) CreateExchangeAndQueue(exchangeName, queueName, bindingKey string) (*amqp.Channel, error) {
	ch, err := c.amqpConn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "Error amqpConn.Channel")
	}

	c.logger.Infof("Declaring exchange: %s", exchangeName)
	err = ch.ExchangeDeclare(
		exchangeName,
		exchangeKind,
		exchangeDurable,
		exchangeAutoDelete,
		exchangeInternal,
		exchangeNoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error ch.ExchangeDeclare")
	}

	queue, err := ch.QueueDeclare(
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

	c.logger.Infof("Declared queue, binding it to exchange: Queue: %v, messagesCount: %v, "+
		"consumerCount: %v, exchange: %v, bindingKey: %v",
		queue.Name,
		queue.Messages,
		queue.Consumers,
		exchangeName,
		bindingKey,
	)

	err = ch.QueueBind(
		queue.Name,
		bindingKey,
		exchangeName,
		queueNoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error ch.QueueBind")
	}

	err = ch.Qos(
		prefetchCount,  // prefetch count
		prefetchSize,   // prefetch size
		prefetchGlobal, // global
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error  ch.Qos")
	}

	return ch, nil
}

func (c *consumerImpl) startConsume(
	ctx context.Context,
	worker func(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery),
	workerPoolSize int,
	queueName string,
	consumerTag string,
) error {
	ch, err := c.amqpConn.Channel()
	if err != nil {
		return errors.Wrap(err, "c.amqpConn.Channel")
	}

	deliveries, err := ch.Consume(
		queueName,
		consumerTag,
		consumeAutoAck,
		consumeExclusive,
		consumeNoLocal,
		consumeNoWait,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "ch.Consume")
	}

	wg := &sync.WaitGroup{}

	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go worker(ctx, wg, deliveries)
	}

	chanErr := <-ch.NotifyClose(make(chan *amqp.Error))
	c.logger.Errorf("ch.NotifyClose: %v", chanErr)

	wg.Wait()

	return chanErr
}

func (c *consumerImpl) RunConsumers(ctx context.Context, cancel context.CancelFunc) {
	go func() {
		if err := c.startConsume(
			ctx,
			c.imagesWorker,
			AvatarsWorkers,
			AvatarsQueueName,
			AvatarsConsumerTag,
		); err != nil {
			c.logger.Errorf("StartResizeConsumer: %v", err)
			cancel()
		}
	}()

}

func (c *consumerImpl) imagesWorker(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery) {
	defer wg.Done()

	for delivery := range messages {
		span, ctx := opentracing.StartSpanFromContext(ctx, "ImageConsumer.resizeWorker")

		c.logger.Infof("processDeliveries deliveryTag% v", delivery.DeliveryTag)

		c.incomingMessages.Inc()

		err := c.service.UpdateUploadedAvatar(ctx, delivery)
		if err != nil {
			if err := delivery.Reject(false); err != nil {
				c.logger.Errorf("Err delivery.Reject: %v", err)
			}
			c.logger.Errorf("Failed to process delivery: %v", err)
			c.errorMessages.Inc()
		} else {
			err = delivery.Ack(false)
			if err != nil {
				c.logger.Errorf("Failed to acknowledge delivery: %v", err)
				c.errorMessages.Inc()
				continue
			}
			c.successMessages.Inc()
		}
		span.Finish()
	}

	c.logger.Info("Deliveries channel closed")
}
