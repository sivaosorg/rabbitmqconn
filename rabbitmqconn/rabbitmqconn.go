package rabbitmqconn

import (
	"fmt"
	"os"

	"github.com/sivaosorg/govm/dbx"
	"github.com/sivaosorg/govm/logger"
	"github.com/sivaosorg/govm/rabbitmqx"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	instance *RabbitMq
	_logger  = logger.NewLogger()
)

func NewRabbitMq() *RabbitMq {
	r := &RabbitMq{}
	return r
}

func (r *RabbitMq) SetConn(value *amqp.Connection) *RabbitMq {
	r.conn = value
	return r
}

func (r *RabbitMq) SetChannel(value *amqp.Channel) *RabbitMq {
	r.channel = value
	return r
}

func NewClient(config rabbitmqx.RabbitMqConfig) (*RabbitMq, dbx.Dbx) {
	s := dbx.NewDbx().SetDebugMode(config.DebugMode)
	if !config.IsEnabled {
		s.SetConnected(false).
			SetMessage("RabbitMQ unavailable").
			SetError(fmt.Errorf(s.Message))
		return &RabbitMq{}, *s
	}
	if instance != nil {
		s.SetConnected(true).SetNewInstance(false)
		return instance, *s
	}
	conn, err := amqp.Dial(config.ToUrlConn())
	if err != nil {
		s.SetConnected(false).SetError(err).SetMessage(err.Error())
		return &RabbitMq{}, *s
	}
	channel, err := conn.Channel()
	if err != nil {
		s.SetConnected(false).SetError(err).SetMessage(err.Error())
		return &RabbitMq{}, *s
	}
	if config.DebugMode {
		_logger.Info(fmt.Sprintf("RabbitMQ client connection:: %s", config.Json()))
		_logger.Info(fmt.Sprintf("Connected successfully to rabbitmq:: %s", config.ToUrlConn()))
	}
	pid := os.Getpid()
	s.SetConnected(true).SetMessage("Connection established").SetPid(pid).SetNewInstance(true)
	instance = NewRabbitMq().SetConn(conn).SetChannel(channel)
	return instance, *s
}

func (c *RabbitMq) Close() {
	c.channel.Close()
	c.conn.Close()
}