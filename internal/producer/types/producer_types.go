package producer_types

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel interface {
	PublishWithContext(ctx context.Context, exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error
	Close() error
}
