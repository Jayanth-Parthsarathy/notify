package consumer

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	consumer_types "github.com/jayanth-parthsarathy/notify/internal/consumer/types"
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

func retry(ch consumer_types.Channel, d consumer_types.Delivery, retryCount int) {
	log.Debugf("This is the %d attempt", retryCount)
	retryQueueName := getRetryQueueName(retryCount)
	if retryCount >= 3 {
		log.Warnf("Max retries reached. Sending to DLQ: %s", d.Body())
		err := d.Nack(false, false)
		logs.LogError(err, "Was not able to nack in retry")
		return
	}
	log.Debugf("This is the %d attempt going to %s queue", retryCount, retryQueueName)
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
			MessageId:    d.MessageId(),
		},
	)
	logs.LogError(err, "Failed to retry")
}

func processMessage(d consumer_types.Delivery, ch consumer_types.Channel, em consumer_types.EmailSender) {
	retryCount := getRetryCount(d.Headers())
	var reqBody types.RequestBody
	log.Debugf("Message received from consumer or retry_queue: %s", d.Body())
	err := json.Unmarshal(d.Body(), &reqBody)
	logs.LogError(err, "Error with unmarshalling json")
	if err != nil {
		_ = d.Nack(false, false)
		return
	}
	err = em.SendEmail(reqBody.Email, reqBody.Message, reqBody.Subject)
	logs.LogError(err, "Failed to send email")
	if err != nil {
		if _, ok := err.(*consumer_types.InvalidEmailError); ok {
			nackErr := d.Nack(false, false)
			logs.LogError(nackErr, "Failed to nack on invalid email")
		} else {
			retry(ch, d, retryCount+1)
		}
		return
	}
	log.Debugf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
	err = d.Ack(false)
	logs.LogError(err, "Not able to acknowledge:")
	if err != nil {
		err = d.Nack(false, true)
		logs.LogError(err, "Failed to nack")
		return
	}
}

func workerConsumeAndProcessMessage(ch *amqp.Channel, id int, em consumer_types.EmailSender) {
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
		log.Debugf("Worker %d: Started processing message", id)
		processMessage(consumer_types.NewDeliveryAdapter(d), ch, em)
		log.Debugf("Worker %d: Finished processing message", id)
	}
}

func worker(id int, conn *amqp.Connection, wg *sync.WaitGroup, em consumer_types.EmailSender) {
	ch := util.CreateChannel(conn)
	defer ch.Close()
	defer wg.Done()
	workerConsumeAndProcessMessage(ch, id, em)
}

func dlqConsumeAndProcessMessages(ch *amqp.Channel, id int, db *pgx.Conn) {
	err := ch.Qos(1, 0, false)
	logs.LogError(err, "Failed to set qos for channel")
	f, err := os.OpenFile("nack.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	dlqMsgs, err := ch.Consume(constants.DLQName, "", false, false, false, false, nil)
	logs.FailOnError(err, "Failed to read messages")
	logs.LogError(err, "Failed to open log file")
	for d := range dlqMsgs {
		log.Debugf("DLQ Worker %d: Started processing message", id)
		err = processDLQMessage(consumer_types.NewDeliveryAdapter(d), f, db)
		logs.LogError(err, "Error with processDLQMessage")
		if err == nil {
			log.Debugf("DLQ Worker %d: Finished processing message", id)
		}
	}
}

func dlqWorker(id int, conn *amqp.Connection, wg *sync.WaitGroup, db *pgx.Conn) {
	ch := util.CreateChannel(conn)
	defer ch.Close()
	defer wg.Done()
	dlqConsumeAndProcessMessages(ch, id, db)
}

func processDLQMessage(d consumer_types.Delivery, f *os.File, db consumer_types.DBExecutor) error {
	bodyJson := d.Body()
	headersMap := make(map[string]interface{})
	for k, v := range d.Headers() {
		headersMap[k] = v
	}
	_, err := db.Exec(context.Background(),
		`INSERT INTO dlq_messages (message_id, body, headers) VALUES ($1, $2, $3)`,
		d.MessageId(),
		bodyJson,
		headersMap,
	)
	if err != nil {
		logs.LogError(err, "Failed to insert DLQ message into DB")
		_ = d.Nack(false, true)
		return err
	}
	err = d.Ack(false)
	logs.LogError(err, "Failed to Ack DLQ message")
	if err != nil {
		d.Nack(false, true)
		return err
	}
	return nil
}

func StartWorkers(conn *amqp.Connection, emailSender consumer_types.EmailSender, db *pgx.Conn) {
	numWorkers := 5
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, conn, &wg, emailSender)
	}
	wg.Add(1)
	go dlqWorker(numWorkers+1, conn, &wg, db)
	wg.Wait()
}
