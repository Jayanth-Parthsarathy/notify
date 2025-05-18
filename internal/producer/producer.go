package producer

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handleNotification(w http.ResponseWriter, req *http.Request, ch *amqp.Channel, q *amqp.Queue) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only post method is accepted", http.StatusMethodNotAllowed)
		return
	}
	var reqBody types.RequestBody
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
			ContentType:  "application/json",
			Body:         jsonBody,
		})
		logs.LogError(err, "Failed to publish message:")
		log.Printf("Published message: %s\n", string(jsonBody))
	}()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification queued successfully"))
}

func StartServer(q *amqp.Queue, ch *amqp.Channel) {
	http.HandleFunc("/notify", func(w http.ResponseWriter, req *http.Request) {
		handleNotification(w, req, ch, q)
	})
	err := http.ListenAndServe(":8090", nil)
	logs.FailOnError(err, "Server failed to start")
}
