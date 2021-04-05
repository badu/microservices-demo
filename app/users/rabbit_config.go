package users

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

	UserExchange = "users"

	AvatarsQueueName   = "avatars_queue"
	AvatarsConsumerTag = "user_avatar_consumer"
	AvatarsWorkers     = 5
	AvatarsBindingKey  = "update_avatar_key"
)
