package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jayanth-parthsarathy/notify/common"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s : %s", msg, err)
	}
}

func logError(err error, msg string) {
	if err != nil {
		log.Printf("%s : %s", msg, err)
	}
}

func handleNotification(w http.ResponseWriter, req *http.Request, ch *amqp.Channel, q *amqp.Queue) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only post method is accepted", http.StatusMethodNotAllowed)
		return
	}
	var reqBody common.RequestBody
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Invalid JSON Structure", http.StatusInternalServerError)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = ch.PublishWithContext(ctx, "", q.Name, false, false, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType: "application/json",
			Body:        jsonBody,
		})
		logError(err, "Failed to publish message:")
		log.Printf("Published message: %s\n", string(jsonBody))
	}()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification queued successfully"))
}

func main() {
	err := godotenv.Load(".env")
	failOnError(err, "Failed to load .env")
	rabbitmqUrl := os.Getenv("RABBIT_MQ_URL")
	conn, err := amqp.Dial(rabbitmqUrl)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to create channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"notification_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to create queue")
	http.HandleFunc("/notify", func(w http.ResponseWriter, req *http.Request) {
		handleNotification(w, req, ch, &q)
	})
	err = http.ListenAndServe(":8090", nil)
	failOnError(err, "Server failed to start")
}
