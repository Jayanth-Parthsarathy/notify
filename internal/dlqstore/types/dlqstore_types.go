package dlqstore_types

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DeadLetterMessage struct {
	ID      string
	Headers amqp.Table
	Payload string
	Type    string
	Raw     []byte
}

type DLQInspector interface {
	ListMessages(limit int) ([]DeadLetterMessage, error)
	RequeueMessage(messageId string) error
}

type RequestBody struct {
	MessageId string
}

type AMQPChannel interface {
	Get(queue string, autoAck bool) (amqp.Delivery, bool, error)
	Nack(tag uint64, multiple, requeue bool) error
	Ack(tag uint64, multiple bool) error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Close() error
}

type AMQPConnection interface {
	Channel() (AMQPChannel, error)
}

type realAMQPConnection struct {
	conn *amqp.Connection
}

type realAMQPChannel struct {
	ch *amqp.Channel
}

func NewConnectionAdapter(conn *amqp.Connection) *realAMQPConnection {
	return &realAMQPConnection{conn: conn}
}

func (r *realAMQPConnection) Channel() (AMQPChannel, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	return &realAMQPChannel{ch: ch}, nil
}

func (c *realAMQPChannel) Get(queue string, autoAck bool) (amqp.Delivery, bool, error) {
	return c.ch.Get(queue, autoAck)
}

func (c *realAMQPChannel) Nack(tag uint64, multiple, requeue bool) error {
	return c.ch.Nack(tag, multiple, requeue)
}
func (c *realAMQPChannel) Ack(tag uint64, multiple bool) error {
	return c.ch.Ack(tag, multiple)
}

func (c *realAMQPChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.ch.Publish(exchange, key, mandatory, immediate, msg)
}
func (c *realAMQPChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return c.ch.Qos(prefetchCount, prefetchSize, global)
}
func (c *realAMQPChannel) Close() error {
	return c.ch.Close()
}

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
