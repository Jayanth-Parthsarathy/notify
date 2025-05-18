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

func DeclareQueue(ch *amqp.Channel) *amqp.Queue {
	q, err := ch.QueueDeclare("notification_queue", true, false, false, false, nil)
	logs.FailOnError(err, "Failed to create queue")
	return &q
}
