package consumer


import (
	"encoding/json"
	"log"
	"sync"
	logs "github.com/jayanth-parthsarathy/notify/internal/common/log"
	types "github.com/jayanth-parthsarathy/notify/internal/common/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func ProcessMessage(d amqp.Delivery) {
	var reqBody types.RequestBody
	log.Printf("some message received: %s", d.Body)
	err := json.Unmarshal(d.Body, &reqBody)
	if err != nil {
		log.Print("Error with unmarshalling json")
		return
	}
	log.Printf("The message recipient is: %s and the message is: %s", reqBody.Email, reqBody.Message)
	err = d.Ack(false)
	if err != nil {
		log.Printf("Not able to acknowledge: %s", err)
		return
	}
}

func Worker(id int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()
	for d := range msgs {
		// time.Sleep(time.Duration(3 * time.Second))
		log.Printf("Worker %d: Started processing message", id)
		ProcessMessage(d)
		log.Printf("Worker %d: Finished processing message", id)
	}
}

func StartWorkers(ch *amqp.Channel, q *amqp.Queue) {
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	logs.FailOnError(err, "Failed to read messages")
	numWorkers := 5
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go Worker(i, msgs, &wg)
	}
	wg.Wait()
}
