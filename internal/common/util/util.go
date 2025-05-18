package util

import (
	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	logs.LogError(err, "Failed to load .env")
}

func ConnectToRabbitMQ() *amqp.Connection {
	rabbitmqUrl := os.Getenv("RABBIT_MQ_URL")
	conn, err := amqp.Dial(rabbitmqUrl)
	logs.FailOnError(err, "Failed to connect to RabbitMQ")
	return conn
}

func CreateChannel(conn *amqp.Connection) *amqp.Channel {
	ch, err := conn.Channel()
	logs.FailOnError(err, "Failed to create channel")
	return ch
}

func declareExchange(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
		constants.RetryExchangeName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func DeclareQueue(conn *amqp.Connection) {
	ch, err := conn.Channel()
	logs.FailOnError(err, "Failed to create channel")
	defer ch.Close()
	err = declareExchange(ch)
	logs.FailOnError(err, "Failed to declare exchange")
	mainQueueArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": constants.DLQName,
	}
	retry10sArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": constants.MainQueueName,
		"x-message-ttl":             int32(10000),
	}
	retry30sArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": constants.MainQueueName,
		"x-message-ttl":             int32(30000),
	}
	retry60sArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": constants.MainQueueName,
		"x-message-ttl":             int32(60000),
	}

	_, err = ch.QueueDeclare(
		constants.MainQueueName, // name of the queue
		true,                    // durable
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		mainQueueArgs,           // arguments
	)
	logs.FailOnError(err, "Failed to create main_queue")
	logs.FailOnError(err, "Failed to bind retry_queue with retry_exchange")
	_, err = ch.QueueDeclare(
		constants.DLQName,
		true,
		false,
		false,
		false,
		nil,
	)
	logs.FailOnError(err, "Failed to declare dlq")
	_, err = ch.QueueDeclare(
		constants.Retry10sQueue,
		true,
		false,
		false,
		false,
		retry10sArgs,
	)
	logs.FailOnError(err, "Failed to declare retry-10s")
	_, err = ch.QueueDeclare(
		constants.Retry30sQueue,
		true,
		false,
		false,
		false,
		retry30sArgs,
	)
	logs.FailOnError(err, "Failed to declare retry-30s")
	_, err = ch.QueueDeclare(
		constants.Retry60sQueue,
		true,
		false,
		false,
		false,
		retry60sArgs,
	)
	logs.FailOnError(err, "Failed to declare retry-60s")
	err = ch.QueueBind(
		constants.Retry10sQueue,
		constants.Retry10sQueue,
		constants.RetryExchangeName,
		false,
		nil,
	)
	logs.FailOnError(err, "Failed to bind retry-10s with retry_exchange")
	err = ch.QueueBind(
		constants.Retry30sQueue,
		constants.Retry30sQueue,
		constants.RetryExchangeName,
		false,
		nil,
	)
	logs.FailOnError(err, "Failed to bind retry-30s with retry_exchange")
	err = ch.QueueBind(
		constants.Retry60sQueue,
		constants.Retry60sQueue,
		constants.RetryExchangeName,
		false,
		nil,
	)
	logs.FailOnError(err, "Failed to bind retry-60s with retry_exchange")
}
