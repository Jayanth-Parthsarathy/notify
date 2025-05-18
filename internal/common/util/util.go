package util

import (
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	logs.FailOnError(err, "Failed to load .env")
}

func ConnectToRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	rabbitmqUrl := os.Getenv("RABBIT_MQ_URL")
	conn, err := amqp.Dial(rabbitmqUrl)
	logs.FailOnError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	logs.FailOnError(err, "Failed to create channel")
	return conn, ch
}

func DeclareQueue(ch *amqp.Channel) (*amqp.Queue, *amqp.Queue) {
	dlq, err := ch.QueueDeclare(
		"notification_dlq", // name of the DLQ
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	logs.FailOnError(err, "Failed to declare DLQ")
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": "notification_dlq",
		"x-message-ttl":             int32(60000),
	}
	q, err := ch.QueueDeclare("notification_queue_with_dlq", true, false, false, false, args)
	logs.FailOnError(err, "Failed to create queue")
	return &q, &dlq
}
