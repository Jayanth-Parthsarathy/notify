package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/jayanth-parthsarathy/notify/common"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s : %s", msg, err)
	}
}

func processMessage(d amqp.Delivery) {
	var reqBody common.RequestBody
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

func worker(id int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()
	for d := range msgs {
		// time.Sleep(time.Duration(3 * time.Second))
		log.Printf("Worker %d: Started processing message", id)
		processMessage(d)
		log.Printf("Worker %d: Finished processing message", id)
	}
}

func main() {
	err := godotenv.Load(".env")
	failOnError(err, "Failed to load .env")
	rabbitmqUrl := os.Getenv("RABBIT_MQ_URL")
	conn, err := amqp.Dial(rabbitmqUrl)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to create channel")
	defer ch.Close()
	q, err := ch.QueueDeclare("notification_queue", true, false, false, false, nil)
	failOnError(err, "Failed to create queue")
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to read messages")
	numWorkers := 5
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, msgs, &wg)
	}
	wg.Wait()
}
