package consumer_types

import (
	"context"
	"fmt"
	"net/smtp"
	"os"

	"github.com/jackc/pgconn"
	consumer_util "github.com/jayanth-parthsarathy/notify/internal/consumer/util"
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

type InvalidEmailError struct {
	Email   string
	Message string
}

func (e *InvalidEmailError) Error() string {
	return fmt.Sprintf("%s - %s", e.Email, e.Message)
}

type GmailSender struct {
}

func (g *GmailSender) SendEmail(recipient string, body string, subject string) error {
	if !consumer_util.Valid(recipient) {
		return &InvalidEmailError{Email: recipient, Message: "Invalid email sending it to DLQ"}
	}
	from := os.Getenv("FROM_EMAIL")
	password := os.Getenv("APP_PASSWORD")
	to := []string{recipient}

	smtpHost := os.Getenv("SMTPHOST")
	smtpPort := os.Getenv("SMTPPORT")

	message := []byte(
		"From: " + from + "\r\n" +
			"To: " + recipient + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\";\r\n" +
			"\r\n" +
			body + "\r\n")

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	return err
}

type DBExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
