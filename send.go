package main

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
	"time"
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

func handleNotification(w http.ResponseWriter, req *http.Request, ch *amqp.Channel, q amqp.Queue) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only post method is accepted", http.StatusMethodNotAllowed)
		return
	}
	var reqBody RequestBody
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = ch.PublishWithContext(ctx, "", q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonBody,
	})
	if err != nil {
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}
	log.Printf("Published message: %s\n", string(jsonBody))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification queued successfully"))
	return
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to create channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"notification",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to create queue")
	http.HandleFunc("/notify", func(w http.ResponseWriter, req *http.Request) {
		handleNotification(w, req, ch, q)
	})
	failOnError(err, "Failed to publish a message")
	http.ListenAndServe(":8090", nil)
}
