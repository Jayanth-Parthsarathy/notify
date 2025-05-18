package producer

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	amqp "github.com/rabbitmq/amqp091-go"
)

func validateRequestBody(w http.ResponseWriter, req *http.Request) []byte {
	var reqBody types.RequestBody
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil
	}
	defer req.Body.Close()
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Invalid JSON Structure", http.StatusInternalServerError)
		return nil
	}
	return jsonBody
}

func publishMessage(jsonBody []byte, ch *amqp.Channel, w http.ResponseWriter, ctx context.Context) error {
	err := ch.PublishWithContext(ctx, "", constants.MainQueueName, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonBody,
	})
	if err != nil {
		logs.LogError(err, "Failed to publish message:")
		http.Error(w, "could not queue notification", http.StatusInternalServerError)
		return err
	}
	return nil
}

func writeSuccessResponse(w http.ResponseWriter, jsonBody []byte) {
	log.Printf("Published message: %s\n", string(jsonBody))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification queued successfully"))
}

func handleNotification(w http.ResponseWriter, req *http.Request, conn *amqp.Connection) {
	ch := util.CreateChannel(conn)
	defer ch.Close()
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	if req.Method != http.MethodPost {
		http.Error(w, "Only post method is accepted", http.StatusMethodNotAllowed)
		return
	}
	jsonBody := validateRequestBody(w, req)
	if jsonBody == nil {
		return
	}
	err := publishMessage(jsonBody, ch, w, ctx)
	if err != nil {
		return
	}
	writeSuccessResponse(w, jsonBody)
}

func StartServer(conn *amqp.Connection) {
	http.HandleFunc("/notify", func(w http.ResponseWriter, req *http.Request) {
		handleNotification(w, req, conn)
	})
	err := http.ListenAndServe(":8090", nil)
	logs.FailOnError(err, "Server failed to start")
}
