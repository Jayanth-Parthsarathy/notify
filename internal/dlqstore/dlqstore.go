package dlqstore

import (
	"context"
	"fmt"

	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v4"
	"github.com/jayanth-parthsarathy/notify/internal/dlqstore/types"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	common "github.com/jayanth-parthsarathy/notify/internal/common/log"
)

type PostgresInspector struct {
	Conn        dlqstore_types.AMQPConnection
	RequeueName string
	DB          dlqstore_types.DB
}

func (p *PostgresInspector) ListMessages(limit int) ([]dlqstore_types.DeadLetterMessage, error) {
	rows, err := p.DB.Query(context.Background(), `SELECT message_id, headers, body FROM dlq_messages ORDER BY received_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []dlqstore_types.DeadLetterMessage
	for rows.Next() {
		var msg dlqstore_types.DeadLetterMessage
		var headers, body []byte
		err := rows.Scan(&msg.ID, &headers, &body)
		if err != nil {
			return nil, err
		}
		var headersMap amqp.Table
		if err := json.Unmarshal(headers, &headersMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
		msg.Headers = headersMap
		msg.Payload = string(body)
		messages = append(messages, msg)
	}
	return messages, nil
}

func (p *PostgresInspector) RequeueMessage(messageId string) error {
	ch, err := p.Conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	var headersJSON, body []byte
	err = p.DB.QueryRow(context.Background(),
		`SELECT headers, body FROM dlq_messages WHERE message_id = $1`, messageId,
	).Scan(&headersJSON, &body)
	if err != nil {
		return fmt.Errorf("failed to fetch message from db: %w", err)
	}

	var headers amqp.Table
	if err := json.Unmarshal(headersJSON, &headers); err != nil {
		return fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	clean := make(amqp.Table, len(headers))
	for k, v := range headers {
		if k == "x-death" {
			continue
		}
		clean[k] = v
	}

	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Headers:      clean,
		DeliveryMode: amqp.Persistent,
		MessageId:    messageId,
	}

	if err := ch.Publish("", p.RequeueName, false, false, pub); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	_, err = p.DB.Exec(context.Background(), `DELETE FROM dlq_messages WHERE message_id = $1`, messageId)
	if err != nil {
		return fmt.Errorf("failed to delete message after requeue: %w", err)
	}

	logrus.Infof("Message %s requeued successfully", messageId)
	return nil
}

func NewPgInspector(db dlqstore_types.DB, conn dlqstore_types.AMQPConnection, requeueName string) *PostgresInspector {
	return &PostgresInspector{DB: db, Conn: conn, RequeueName: requeueName}
}

func handleInspect(w http.ResponseWriter, _ *http.Request, inspector dlqstore_types.DLQInspector) {
	msgs, err := inspector.ListMessages(10)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}
	for _, msg := range msgs {
		fmt.Printf("ID: %s\nType: %s\nHeaders: %v\nPayload: %s\n\n",
			msg.ID, msg.Type, msg.Headers, msg.Payload)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func handleRequeue(w http.ResponseWriter, req *http.Request, inspector dlqstore_types.DLQInspector) {
	var reqBody dlqstore_types.RequestBody
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		common.LogError(err, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := inspector.RequeueMessage(reqBody.MessageId); err != nil {
		common.LogError(err, "Failed to requeue")
		http.Error(w, "Failed to requeue", http.StatusInternalServerError)
		return
	}
	logrus.Debugf("Message requeued: %s", reqBody.MessageId)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification requeued successfully"))
}

func StartServer(conn *amqp.Connection, inspector dlqstore_types.DLQInspector, db *pgx.Conn) {
	defer conn.Close()
	http.HandleFunc("/inspect", func(w http.ResponseWriter, req *http.Request) {
		handleInspect(w, req, inspector)
	})
	http.HandleFunc("/requeue", func(w http.ResponseWriter, req *http.Request) {
		handleRequeue(w, req, inspector)
	})
	err := http.ListenAndServe(":8091", nil)
	common.FailOnError(err, "Server failed to start")
}
