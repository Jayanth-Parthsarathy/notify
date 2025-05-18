package consumer

import (
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"sync"

	"net/mail"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	consumer_types "github.com/jayanth-parthsarathy/notify/internal/consumer/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

type InvalidEmailError struct {
	email   string
	message string
}

func (e *InvalidEmailError) Error() string {
	return fmt.Sprintf("%s - %s", e.email, e.message)
}

type GmailSender struct {
}

func (g *GmailSender) SendEmail(recipient string, body string, subject string) error {
	if !valid(recipient) {
		return &InvalidEmailError{email: recipient, message: "Invalid email sending it to DLQ"}
	}
	from := os.Getenv("FROM_EMAIL")
	password := os.Getenv("APP_PASSWORD")
	to := []string{recipient}

	smtpHost := os.Getenv("SMTPHOST")
	smtpPort := os.Getenv("SMTPPORT")

	message := []byte(
		"From: " + from + "\r\n" +
			"To: " + recipient + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\";\r\n" +
			"\r\n" +
			body + "\r\n")

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	return err
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	switch v := headers["x-retry-count"].(type) {
	case int32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func getRetryQueueName(retryCount int) string {
	var retryQueueName string
	switch retryCount {
	case 1:
		retryQueueName = constants.Retry10sQueue
	case 2:
		retryQueueName = constants.Retry30sQueue
	case 3:
		retryQueueName = constants.Retry60sQueue
	default:
		retryQueueName = ""
	}
	return retryQueueName
}

func populateHeader(headers amqp.Table, retryCount int) amqp.Table {
	if headers == nil {
		headers = amqp.Table{}
	}
	headers["x-retry-count"] = int32(retryCount)
	return headers
}

func retry(ch consumer_types.Channel, d consumer_types.Delivery, retryCount int) {
	log.Printf("This is the %d attempt", retryCount)
	retryQueueName := getRetryQueueName(retryCount)
	if retryCount >= 3 {
		log.Printf("Max retries reached. Sending to DLQ: %s", d.Body())
		err := d.Nack(false, false)
		logs.LogError(err, "Was not able to nack in retry")
		return
	}
	log.Printf("This is the %d attempt going to %s queue", retryCount, retryQueueName)
	headers := populateHeader(d.Headers(), retryCount)
	err := d.Ack(false)
	logs.LogError(err, "Failed to ack")
	err = ch.Publish(
		constants.RetryExchangeName,
		retryQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  d.ContentType(),
			Body:         d.Body(),
			Headers:      headers,
			DeliveryMode: amqp.Persistent,
		},
	)
	logs.LogError(err, "Failed to retry")
}

func processMessage(d consumer_types.Delivery, ch consumer_types.Channel, em consumer_types.EmailSender) {
	retryCount := getRetryCount(d.Headers())
	var reqBody types.RequestBody
	log.Printf("Message received from consumer or retry_queue: %s", d.Body())
	err := json.Unmarshal(d.Body(), &reqBody)
	logs.LogError(err, "Error with unmarshalling json")
	if err != nil {
		_ = d.Nack(false, false)
		return
	}
	err = em.SendEmail(reqBody.Email, reqBody.Message, reqBody.Subject)
	logs.LogError(err, "Failed to send email")
	if err != nil {
		if _, ok := err.(*InvalidEmailError); ok {
			nackErr := d.Nack(false, false)
			logs.LogError(nackErr, "Failed to nack on invalid email")
		} else {
			retry(ch, d, retryCount+1)
		}
		return
	}
	log.Printf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
	err = d.Ack(false)
	logs.LogError(err, "Not able to acknowledge:")
	if err != nil {
		err = d.Nack(false, true)
		logs.LogError(err, "Failed to nack")
		return
	}
}

func workerConsumeAndProcessMessage(ch *amqp.Channel, id int) {
	err := ch.Qos(1, 0, false)
	logs.LogError(err, "Failed to set qos for channel")
	msgs, err := ch.Consume(
		constants.MainQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	logs.FailOnError(err, "Failed to read messages")
	for d := range msgs {
		log.Printf("Worker %d: Started processing message", id)
		gmailSender := GmailSender{}
		processMessage(consumer_types.NewDeliveryAdapter(d), ch, &gmailSender)
		log.Printf("Worker %d: Finished processing message", id)
	}
}

func worker(id int, conn *amqp.Connection, wg *sync.WaitGroup) {
	ch := util.CreateChannel(conn)
	defer ch.Close()
	defer wg.Done()
	workerConsumeAndProcessMessage(ch, id)
}

func dlqConsumeAndProcessMessages(ch *amqp.Channel, id int) {
	err := ch.Qos(1, 0, false)
	logs.LogError(err, "Failed to set qos for channel")
	f, err := os.OpenFile("nack.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	dlqMsgs, err := ch.Consume(constants.DLQName, "", false, false, false, false, nil)
	logs.FailOnError(err, "Failed to read messages")
	logs.LogError(err, "Failed to open log file")
	for d := range dlqMsgs {
		log.Printf("DLQ Worker %d: Started processing message", id)
		err = processDLQMessage(d, f)
		logs.LogError(err, "Error with processDLQMessage")
		if err == nil {
			log.Printf("DLQ Worker %d: Finished processing message", id)
		}
	}
}

func dlqWorker(id int, conn *amqp.Connection, wg *sync.WaitGroup) {
	ch := util.CreateChannel(conn)
	defer ch.Close()
	defer wg.Done()
	dlqConsumeAndProcessMessages(ch, id)
}

func processDLQMessage(d amqp.Delivery, f *os.File) error {
	logger := log.New(f, "DLQ: ", log.LstdFlags|log.Lmsgprefix)
	logger.Printf("Message ID: %s, Body: %s, Headers: %v", d.MessageId, d.Body, d.Headers)
	err := d.Ack(false)
	logs.LogError(err, "Failed to Ack DLQ message")
	if err != nil {
		d.Nack(false, true)
		return err
	}
	return nil
}

func StartWorkers(conn *amqp.Connection) {
	numWorkers := 5
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, conn, &wg)
	}
	wg.Add(1)
	go dlqWorker(numWorkers+1, conn, &wg)
	wg.Wait()
}
