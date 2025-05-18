package consumer

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	amqp "github.com/rabbitmq/amqp091-go"
)

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

func retry(ch *amqp.Channel, d amqp.Delivery, retryCount int) {
	log.Printf("This is the %d attempt", retryCount)
	retryQueueName := getRetryQueueName(retryCount)
	if retryCount >= 3 {
		log.Printf("Max retries reached. Sending to DLQ: %s", d.Body)
		err := d.Nack(false, false)
		logs.LogError(err, "Was not able to nack in retry")
		return
	}
	log.Printf("This is the %d attempt going to %s queue", retryCount, retryQueueName)
	headers := populateHeader(d.Headers, retryCount)
	err := d.Ack(false)
	logs.LogError(err, "Failed to ack")
	err = ch.Publish(
		constants.RetryExchangeName,
		retryQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  d.ContentType,
			Body:         d.Body,
			Headers:      headers,
			DeliveryMode: amqp.Persistent,
		},
	)
	logs.LogError(err, "Failed to retry")
}

func processMessage(d amqp.Delivery, ch *amqp.Channel) {
	retryCount := getRetryCount(d.Headers)
	var reqBody types.RequestBody
	log.Printf("Message received from consumer or retry_queue: %s", d.Body)
	err := json.Unmarshal(d.Body, &reqBody)
	logs.LogError(err, "Error with unmarshalling json")
	if err != nil {
		_ = d.Nack(false, false)
		return
	}
	// some email sending processing will happen here, if it fails we retry else we dont (if it fail err wont be nil and so will be retried)
	// err = errors.New("some")
	err = nil
	if err != nil {
		retry(ch, d, retryCount+1)
		return
	}
	log.Printf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
	err = d.Ack(false)
	logs.LogError(err, "Not able to acknowledge:")
	if err != nil {
		d.Nack(false, true)
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
		processMessage(d, ch)
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
