package main

import (
	"encoding/json"
	"log"

	ampq "github.com/rabbitmq/amqp091-go"
)

type RequestBody struct {
	Email   string `json:"email"`
	Message string `json:"message"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s : %s", msg, err)
	}
}

func processMessages(msgs <-chan ampq.Delivery) {
	log.Print("started processsing")
	for d := range msgs {
		var reqBody RequestBody
		log.Printf("some message received: %s", d.Body)
		err := json.Unmarshal(d.Body, &reqBody)
		if err != nil {
			log.Print("Error with unmarshalling json")
			continue
		}
		log.Printf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
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
