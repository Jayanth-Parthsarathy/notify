package producer

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	producer_types "github.com/jayanth-parthsarathy/notify/internal/producer/types"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

func validateRequestBody(w http.ResponseWriter, req *http.Request) []byte {
	var reqBody types.RequestBody
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		logs.LogError(err, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil
	}
	defer req.Body.Close()
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		logs.LogError(err, "Invalid JSON Structure")
		http.Error(w, "Invalid JSON Structure", http.StatusInternalServerError)
		return nil
	}
	return jsonBody
}

func publishMessage(jsonBody []byte, ch producer_types.Channel, w http.ResponseWriter, ctx context.Context) error {
	err := ch.PublishWithContext(ctx, "", constants.MainQueueName, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonBody,
		MessageId:    uuid.New().String(),
	})
	if err != nil {
		logs.LogError(err, "Failed to publish message:")
		http.Error(w, "could not queue notification", http.StatusInternalServerError)
		return err
	}
	return nil
}

func writeSuccessResponse(w http.ResponseWriter, jsonBody []byte) {
	log.Debugf("Published message: %s\n", string(jsonBody))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification queued successfully"))
}

func handleNotification(w http.ResponseWriter, req *http.Request, ch producer_types.Channel) {
	defer ch.Close()
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	if req.Method != http.MethodPost {
		log.Errorf("Method with post is accepted: %s", req.Method)
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
		ch := util.CreateChannel(conn)
		handleNotification(w, req, ch)
	})
	err := http.ListenAndServe(":8090", nil)
	logs.FailOnError(err, "Server failed to start")
}
