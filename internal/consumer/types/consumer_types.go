package consumer_types

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel interface {
	Publish(exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error
}

type Delivery interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
	Body() []byte
	ContentType() string
	Headers() amqp.Table
	MessageId() string
}

type DeliveryAdapter struct {
	D amqp.Delivery
}

func (a *DeliveryAdapter) Ack(multiple bool) error           { return a.D.Ack(multiple) }
func (a *DeliveryAdapter) Nack(multiple, requeue bool) error { return a.D.Nack(multiple, requeue) }
func (a *DeliveryAdapter) Body() []byte                      { return a.D.Body }
func (a *DeliveryAdapter) ContentType() string               { return a.D.ContentType }
func (a *DeliveryAdapter) Headers() amqp.Table               { return a.D.Headers }
func (a *DeliveryAdapter) MessageId() string                 { return a.D.MessageId }

func NewDeliveryAdapter(d amqp.Delivery) *DeliveryAdapter {
	return &DeliveryAdapter{D: d}
}

type EmailSender interface {
	SendEmail(recipient string, body string, subject string) error
}
