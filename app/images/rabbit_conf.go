package images

import (
	"errors"

	"github.com/streadway/amqp"
)

const (
	exchangeKind       = "direct"
	exchangeDurable    = true
	exchangeAutoDelete = false
	exchangeInternal   = false
	exchangeNoWait     = false

	queueDurable    = true
	queueAutoDelete = false
	queueExclusive  = false
	queueNoWait     = false

	publishMandatory = false
	publishImmediate = false

	prefetchCount  = 1
	prefetchSize   = 0
	prefetchGlobal = false

	consumeAutoAck   = false
	consumeExclusive = false
	consumeNoLocal   = false
	consumeNoWait    = false

	ExchangeName = "images"

	ResizeQueueName   = "resize_queue"
	ResizeConsumerTag = "resize_consumer"
	ResizeWorkers     = 10
	ResizeBindingKey  = "resize_image_key"

	CreateQueueName   = "create_queue"
	CreateConsumerTag = "create_consumer"
	CreateWorkers     = 5
	CreateBindingKey  = "create_image_key"

	UploadHotelImageQueue       = "upload_hotel_image_queue"
	UploadHotelImageConsumerTag = "upload_hotel_image_consumer_tag"
	UploadHotelImageWorkers     = 10
	UploadHotelImageBindingKey  = "upload_hotel_image_binding_key"
)

// Initialize consumers
func (c *ConsumerImpl) Initialize() error {
	if err := c.Dial(); err != nil {
		return errors.Join(err, errors.New("images.ConsumerDial"))
	}

	updateImageChan, err := c.CreateExchangeAndQueue(ExchangeName, UploadHotelImageQueue, UploadHotelImageBindingKey)
	if err != nil {
		return errors.Join(err, errors.New("images.CreateExchangeAndQueue"))
	}
	c.channels = append(c.channels, updateImageChan)

	resizeChan, err := c.CreateExchangeAndQueue(ExchangeName, ResizeQueueName, ResizeBindingKey)
	if err != nil {
		return errors.Join(err, errors.New("images.CreateExchangeAndQueue"))
	}
	c.channels = append(c.channels, resizeChan)

	createImgChan, err := c.CreateExchangeAndQueue(ExchangeName, CreateQueueName, CreateBindingKey)
	if err != nil {
		return errors.Join(err, errors.New("images.CreateExchangeAndQueue"))
	}
	c.channels = append(c.channels, createImgChan)

	return nil
}

// CloseChannels close active channels
func (c *ConsumerImpl) CloseChannels() {
	for _, channel := range c.channels {
		go func(ch *amqp.Channel) {
			if err := ch.Close(); err != nil {
				c.logger.Errorf("CloseChannels ch.Close error: %v", err)
			}
		}(channel)
	}
}
