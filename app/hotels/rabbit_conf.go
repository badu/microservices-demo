package hotels

import (
	"github.com/pkg/errors"
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

	ExchangeName = "hotels"

	UpdateImageQueue       = "update_hotel_image"
	UpdateImageBindingKey  = "update_hotel_image_key"
	UpdateImageWorkers     = 5
	UpdateImageConsumerTag = "update_hotel_image_consumer"
)

func (c *consumerImpl) Initialize() error {
	if err := c.Dial(); err != nil {
		return errors.Wrap(err, "Consumer Dial")
	}

	updateImageChan, err := c.CreateExchangeAndQueue(ExchangeName, UpdateImageQueue, UpdateImageBindingKey)
	if err != nil {
		return errors.Wrap(err, "CreateExchangeAndQueue")
	}

	c.channels = append(c.channels, updateImageChan)

	return nil
}

// CloseChannels close active channels
func (c *consumerImpl) CloseChannels() {
	for _, channel := range c.channels {
		go func(ch *amqp.Channel) {
			if err := ch.Close(); err != nil {
				c.logger.Errorf("CloseChannels ch.Close error: %v", err)
			}
		}(channel)
	}
}
