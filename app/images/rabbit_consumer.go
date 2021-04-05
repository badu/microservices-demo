package images

import (
	"context"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/rabbitmq"
)

type Consumer struct {
	Worker    func(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery)
	PoolSize  int
	QueueName string
	Tag       string
}

type consumerImpl struct {
	amqpConn         *amqp.Connection
	logger           logger.Logger
	cfg              *config.Config
	service          Service
	consumers        []*Consumer
	channels         []*amqp.Channel
	incomingMessages prometheus.Counter
	successMessages  prometheus.Counter
	errorMessages    prometheus.Counter
}

func NewImageConsumer(
	logger logger.Logger,
	cfg *config.Config,
	service Service,
	incomingMessages prometheus.Counter,
	successMessages prometheus.Counter,
	errorMessages prometheus.Counter,
) *consumerImpl {
	return &consumerImpl{
		logger:           logger,
		cfg:              cfg,
		service:          service,
		incomingMessages: incomingMessages,
		successMessages:  successMessages,
		errorMessages:    errorMessages,
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

func (c *consumerImpl) AddConsumer(consumer *Consumer) {
	c.consumers = append(c.consumers, consumer)
}

func (c *consumerImpl) run(ctx context.Context, cancel context.CancelFunc) {
	for _, cs := range c.consumers {
		go func(consumer *Consumer) {
			if err := c.startConsume(
				ctx,
				consumer.Worker,
				consumer.PoolSize,
				consumer.QueueName,
				consumer.Tag,
			); err != nil {
				c.logger.Errorf("StartResizeConsumer: %v", err)
				cancel()
			}
		}(cs)
	}
}

func (c *consumerImpl) RunConsumers(ctx context.Context, cancel context.CancelFunc) {
	c.AddConsumer(&Consumer{
		Worker:    c.resizeWorker,
		PoolSize:  ResizeWorkers,
		QueueName: ResizeQueueName,
		Tag:       ResizeConsumerTag,
	})
	c.AddConsumer(&Consumer{
		Worker:    c.createImageWorker,
		PoolSize:  CreateWorkers,
		QueueName: CreateQueueName,
		Tag:       CreateConsumerTag,
	})
	c.AddConsumer(&Consumer{
		Worker:    c.processHotelImageWorker,
		PoolSize:  UploadHotelImageWorkers,
		QueueName: UploadHotelImageQueue,
		Tag:       UploadHotelImageConsumerTag,
	})
	c.run(ctx, cancel)
}

func (c *consumerImpl) resizeWorker(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery) {
	defer wg.Done()
	for delivery := range messages {
		span, ctx := opentracing.StartSpanFromContext(ctx, "consumerImpl.resizeWorker")

		c.logger.Infof("processDeliveries deliveryTag% v", delivery.DeliveryTag)

		c.incomingMessages.Inc()

		err := c.service.ResizeImage(ctx, delivery)
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
			}
			c.successMessages.Inc()
		}
		span.Finish()
	}

	c.logger.Info("Deliveries channel closed")
}

func (c *consumerImpl) createImageWorker(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery) {
	defer wg.Done()
	for delivery := range messages {
		span, ctx := opentracing.StartSpanFromContext(ctx, "consumerImpl.createImageWorker")

		c.logger.Infof("processDeliveries deliveryTag% v", delivery.DeliveryTag)

		c.incomingMessages.Inc()

		err := c.service.Create(ctx, delivery)
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

func (c *consumerImpl) processHotelImageWorker(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery) {
	defer wg.Done()
	for delivery := range messages {
		span, ctx := opentracing.StartSpanFromContext(ctx, "consumerImpl.createImageWorker")

		c.logger.Infof("processDeliveries deliveryTag% v", delivery.DeliveryTag)

		c.incomingMessages.Inc()

		err := c.service.ProcessHotelImage(ctx, delivery)
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
