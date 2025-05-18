package consumer

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func processMessage(d amqp.Delivery) {
	var reqBody types.RequestBody
	log.Printf("some message received: %s", d.Body)
	err := json.Unmarshal(d.Body, &reqBody)
	if err != nil {
		logs.LogError(err, "Error with unmarshalling json")
		err = d.Nack(false, false)
		if err != nil {
			logs.LogError(err, "Error with Nacking:")
		}
		return
	}
	log.Printf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
	err = d.Ack(false)
	if err != nil {
		logs.LogError(err, "Not able to acknowledge:")
		err = d.Nack(false, false)
		if err != nil {
			logs.LogError(err, "Error with Nacking:")
		}
		return
	}
}

func worker(id int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()
	for d := range msgs {
		// time.Sleep(time.Duration(3 * time.Second))
		log.Printf("Worker %d: Started processing message", id)
		processMessage(d)
		log.Printf("Worker %d: Finished processing message", id)
	}
}

func dlqWorker(id int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup) error {
	defer wg.Done()
	f, err := os.OpenFile("nack.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logs.LogError(err, "Failed to open log file")
		return err
	}
	for d := range msgs {
		// time.Sleep(time.Duration(3 * time.Second))
		log.Printf("DLQ Worker %d: Started processing message", id)
		_ = processDLQMessage(d, f)
		log.Printf("DLQ Worker %d: Finished processing message", id)
	}
	return nil
}

func processDLQMessage(d amqp.Delivery, f *os.File) error {
	logger := log.New(f, "DLQ: ", log.LstdFlags|log.Lmsgprefix)
	logger.Printf("Message ID: %s, Body: %s, Headers: %v", d.MessageId, d.Body, d.Headers)
	err := d.Ack(false)
	if err != nil {
		logger.Printf("Failed to Ack DLQ message: %v\n", err)
		d.Nack(false, true)
		return err
	}
	return nil
}

func StartWorkers(ch *amqp.Channel, q *amqp.Queue, dlq *amqp.Queue) {
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	logs.FailOnError(err, "Failed to read messages")
	dlqMsgs, err := ch.Consume(dlq.Name, "", false, false, false, false, nil)
	logs.FailOnError(err, "Failed to read dlq messages")
	numWorkers := 5
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, msgs, &wg)
	}
	wg.Add(1)
	go dlqWorker(numWorkers+1, dlqMsgs, &wg)
	wg.Wait()
}
