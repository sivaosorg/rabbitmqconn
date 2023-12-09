package rabbitmqconn

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sivaosorg/govm/rabbitmqx"
	"github.com/sivaosorg/govm/utils"
)

type RabbitMqService interface {
	CreateTopic(topic string) error
	RemoveTopic(topic string) error
	Producer(topic string, message interface{}) error
	Consumer(topic, queue string, callback func(next amqp.Delivery)) error
}

type RabbitMqCoreService interface {
	RemoveExchange(exchangeName string) error
	DeclareExchangeConf() error
	DeclareExchangeWith(exchangeName string, exchangeType string, durable bool) error
	DeclareQueueConf() (amqp.Queue, error)
	DeclareQueueWith(queueName string, durable bool) (amqp.Queue, error)
	BindQueueExchangeConf() error
	BindQueueExchangeWith(queueName, exchangeName string) error
	ProduceConf(message interface{}) error
	ProduceWith(exchangeName string, exchangeType string, durable bool, message interface{}) error
	ConsumeConf(callback func(next amqp.Delivery)) error
	ConsumeWith(queueName string, exchangeName, exchangeType string, durable bool, callback func(next amqp.Delivery)) error
}

type rabbitmqCoreServiceImpl struct {
	c *RabbitMq
}

type rabbitmqServiceImpl struct {
	c *RabbitMq
}

func NewRabbitMqService(c *RabbitMq) RabbitMqService {
	s := &rabbitmqServiceImpl{
		c: c,
	}
	return s
}

func NewRabbitMqCoreService(c *RabbitMq) RabbitMqCoreService {
	s := &rabbitmqCoreServiceImpl{
		c: c,
	}
	return s
}

func (r *rabbitmqServiceImpl) CreateTopic(topic string) error {
	err := r.c.channel.ExchangeDeclare(
		topic,               // name exchange
		amqp.ExchangeFanout, // type exchange
		true,                // Durable
		false,               // Auto-deleted
		false,               // Internal
		false,               // No-wait
		nil,
	)
	return err
}

func (r *rabbitmqServiceImpl) RemoveTopic(topic string) error {
	err := r.c.channel.ExchangeDelete(
		topic,
		false,
		false,
	)
	return err
}

func (r *rabbitmqServiceImpl) Producer(topic string, message interface{}) error {
	err := r.c.channel.ExchangeDeclare(
		topic,               // name exchange
		amqp.ExchangeFanout, // type exchange
		true,                // Durable
		false,               // Auto-deleted
		false,               // Internal
		false,               // No-wait
		nil,                 // Arguments
	)
	if err != nil {
		return err
	}
	err = r.c.channel.Publish(
		topic,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json", // application/json, text/plain
			Body:        []byte(utils.ToJson(message)),
		},
	)
	return err
}

func (r *rabbitmqServiceImpl) Consumer(topic, queue string, callback func(next amqp.Delivery)) error {
	err := r.c.channel.ExchangeDeclare(
		topic,               // name exchange
		amqp.ExchangeFanout, // type exchange
		true,                // Durable
		false,               // Auto-deleted
		false,               // Internal
		false,               // No-wait
		nil,                 // Arguments
	)
	if err != nil {
		return err
	}
	q, err := r.c.channel.QueueDeclare(
		queue, // name queue
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}
	err = r.c.channel.QueueBind(
		q.Name, // name queue
		"",     // Routing key
		topic,  // name exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}
	msg, err := r.c.channel.Consume(
		q.Name, // name queue
		"",     // Consumer
		true,   // Auto-acknowledge
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Arguments
	)
	if err != nil {
		return err
	}
	forever := make(chan bool)
	go func() {
		for d := range msg {
			callback(d)
		}
	}()
	_logger.Info("Consumer is waiting for messages (%s)...", topic)
	<-forever
	return nil
}

func (r *rabbitmqCoreServiceImpl) RemoveExchange(exchangeName string) error {
	err := r.c.channel.ExchangeDelete(
		exchangeName,
		false,
		false,
	)
	return err
}

func (r *rabbitmqCoreServiceImpl) DeclareExchangeConf() error {
	rabbitmqx.RabbitMqExchangeConfigValidator(&r.c.Config.Message.Exchange)
	return r.DeclareExchangeWith(r.c.Config.Message.Exchange.Name,
		r.c.Config.Message.Exchange.Kind,
		r.c.Config.Message.Exchange.Durable)
}

func (r *rabbitmqCoreServiceImpl) DeclareExchangeWith(exchangeName string, exchangeType string, durable bool) error {
	config := rabbitmqx.NewRabbitMqExchangeConfig().SetName(exchangeName).SetKind(exchangeType).SetDurable(durable)
	err := r.c.channel.ExchangeDeclare(
		config.Name,    // name exchange
		config.Kind,    // type exchange
		config.Durable, // Durable
		false,          // Auto-deleted
		false,          // Internal
		false,          // No-wait
		nil,
	)
	return err
}

func (r *rabbitmqCoreServiceImpl) DeclareQueueConf() (amqp.Queue, error) {
	rabbitmqx.RabbitMqQueueConfigValidator(&r.c.Config.Message.Queue)
	return r.DeclareQueueWith(r.c.Config.Message.Queue.Name, r.c.Config.Message.Queue.Durable)
}

func (r *rabbitmqCoreServiceImpl) DeclareQueueWith(queueName string, durable bool) (amqp.Queue, error) {
	config := rabbitmqx.NewRabbitMqQueueConfig().SetName(queueName).SetDurable(durable)
	q, err := r.c.channel.QueueDeclare(
		config.Name,    // name queue
		config.Durable, // Durable
		false,          // Delete when unused
		false,          // Exclusive
		false,          // No-wait
		nil,            // Arguments
	)
	return q, err
}

func (r *rabbitmqCoreServiceImpl) BindQueueExchangeConf() error {
	rabbitmqx.RabbitMqExchangeConfigValidator(&r.c.Config.Message.Exchange)
	rabbitmqx.RabbitMqQueueConfigValidator(&r.c.Config.Message.Queue)
	return r.BindQueueExchangeWith(r.c.Config.Message.Queue.Name, r.c.Config.Message.Exchange.Name)
}

func (r *rabbitmqCoreServiceImpl) BindQueueExchangeWith(queueName, exchangeName string) error {
	exchange := rabbitmqx.NewRabbitMqExchangeConfig().SetName(exchangeName)
	queue := rabbitmqx.NewRabbitMqQueueConfig().SetName(queueName)
	err := r.c.channel.QueueBind(
		queue.Name,    // name queue
		"",            // Routing key
		exchange.Name, // name exchange
		false,
		nil,
	)
	return err
}

func (r *rabbitmqCoreServiceImpl) ProduceConf(message interface{}) error {
	if !r.c.Config.Message.IsEnabled {
		return fmt.Errorf("ProduceConf, message unavailable (enabled = false)")
	}
	rabbitmqx.RabbitMqExchangeConfigValidator(&r.c.Config.Message.Exchange)
	return r.ProduceWith(r.c.Config.Message.Exchange.Name,
		r.c.Config.Message.Exchange.Kind,
		r.c.Config.Message.Exchange.Durable,
		message)
}

func (r *rabbitmqCoreServiceImpl) ProduceWith(exchangeName string, exchangeType string, durable bool, message interface{}) error {
	err := r.DeclareExchangeWith(exchangeName, exchangeType, durable)
	if err != nil {
		return err
	}
	if r.c.Config.DebugMode {
		_logger.Info(fmt.Sprintf("Producer is running for messages (exchange: %s) outgoing data: %v", exchangeName, utils.ToJson(message)))
	} else {
		_logger.Info(fmt.Sprintf("Producer is running for messages (exchange: %s)", exchangeName))
	}
	err = r.c.channel.Publish(
		exchangeName,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json", // application/json, text/plain
			Body:        []byte(utils.ToJson(message)),
		},
	)
	return err
}

func (r *rabbitmqCoreServiceImpl) ConsumeConf(callback func(next amqp.Delivery)) error {
	if !r.c.Config.Message.IsEnabled {
		return fmt.Errorf("ConsumeConf, message unavailable (enabled = false)")
	}
	rabbitmqx.RabbitMqExchangeConfigValidator(&r.c.Config.Message.Exchange)
	rabbitmqx.RabbitMqQueueConfigValidator(&r.c.Config.Message.Queue)
	return r.ConsumeWith(r.c.Config.Message.Queue.Name,
		r.c.Config.Message.Exchange.Name,
		r.c.Config.Message.Exchange.Kind,
		r.c.Config.Message.Exchange.Durable,
		callback)
}

func (r *rabbitmqCoreServiceImpl) ConsumeWith(queueName string, exchangeName, exchangeType string, durable bool, callback func(next amqp.Delivery)) error {
	err := r.DeclareExchangeWith(exchangeName, exchangeType, durable)
	if err != nil {
		return err
	}
	q, err := r.DeclareQueueWith(queueName, durable)
	if err != nil {
		return err
	}
	err = r.BindQueueExchangeWith(q.Name, exchangeName)
	if err != nil {
		return err
	}
	msg, err := r.c.channel.Consume(
		q.Name, // name queue
		"",     // Consumer
		true,   // Auto-acknowledge
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Arguments
	)
	if err != nil {
		return err
	}
	forever := make(chan bool)
	go func() {
		for d := range msg {
			if callback != nil {
				callback(d)
			} else {
				callbackDefault(d)
			}
		}
	}()
	_logger.Info(fmt.Sprintf("Consumer is waiting for messages (%s)...", exchangeName))
	<-forever
	return nil
}