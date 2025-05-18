package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"testing"
	"time"

	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/consumer"
	consumer_types "github.com/jayanth-parthsarathy/notify/internal/consumer/types"
	consumer_util "github.com/jayanth-parthsarathy/notify/internal/consumer/util"
	"github.com/jayanth-parthsarathy/notify/internal/producer"
	"github.com/stretchr/testify/assert"
)

type MailHogSender struct {
}

func (m *MailHogSender) SendEmail(recipient string, body string, subject string) error {
	if !consumer_util.Valid(recipient) {
		return &consumer_types.InvalidEmailError{Email: recipient, Message: "Invalid email sending it to DLQ"}
	}
	smtpHost := "localhost"
	smtpPort := "1025"
	from := "test@example.com"
	to := []string{recipient}

	msg := []byte(subject + "\n" + body)

	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		nil,
		from,
		to,
		msg,
	)
	return err
}

func TestIntegration_NotificationFlow(t *testing.T) {
	os.Setenv("RABBIT_MQ_URL", "amqp://guest:guest@localhost:5672/")

	conn := util.ConnectToRabbitMQ()
	if conn == nil {
		t.Fatal("Could not connect to RabbitMQ")
	}
	log.Println("Connection created")
	defer conn.Close()
	util.DeclareQueue(conn)
	log.Println("Queues created")
	go producer.StartServer(conn)
	log.Println("Server started")
	mailHogSender := MailHogSender{}
	go consumer.StartWorkers(conn, &mailHogSender)

	time.Sleep(1 * time.Second)

	// --- Test 1: Success case ---
	t.Run("Success_Email_Should_Not_Be_In_DLQ", func(t *testing.T) {
		payload := map[string]string{
			"email":   "test@integration.com",
			"subject": "Integration Test",
			"message": "This is a test message.",
		}
		body, _ := json.Marshal(payload)

		resp, err := http.Post("http://localhost:8090/notify", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		time.Sleep(3 * time.Second)

		ensureLogExists(t, "failed_notifications.log")
		data, err := ioutil.ReadFile("failed_notifications.log")
		assert.NoError(t, err)
		assert.NotContains(t, string(data), "test@integration.com")
	})

	// --- Test 2: Failure case ---
	t.Run("Invalid_Email_Should_Appear_In_DLQ", func(t *testing.T) {
		payload := map[string]string{
			"email":   "test",
			"subject": "Integration Test",
			"message": "This is a test message.",
		}
		body, _ := json.Marshal(payload)

		resp, err := http.Post("http://localhost:8090/notify", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		time.Sleep(3 * time.Second)

		ensureLogExists(t, "nack.log")
		data, err := ioutil.ReadFile("nack.log")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "test")
	})
}

func ensureLogExists(t *testing.T, filename string) {
	f, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
