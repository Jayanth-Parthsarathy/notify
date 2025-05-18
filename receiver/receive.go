package main

import (
	"log"

	ampq "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s : %s", msg, err)
	}
}

func processMessages(msgs <-chan ampq.Delivery) {
	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)
	}
}

func main() {
	conn, err := ampq.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to create channel")
	defer ch.Close()
	q, err := ch.QueueDeclare("notification", false, false, false, false, nil)
	failOnError(err, "Failed to create queue")
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to read messages")
	forever := make(chan struct{})
	go processMessages(msgs)
	<-forever
}
