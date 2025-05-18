# Notify

This is a simple notification service built using Golang that uses RabbitMQ as a queue, with asynchronous workers to poll and send messages.

RabbitMQ is used to improve the scalability of the system.
Instead of overloading the main server with message processing, we offload messages to a queue, where they are handled asynchronously by worker processes.

This project uses Golang because it is simple to use and easy to get started with.
Additionally, goroutines and channels in Go make implementing concurrency straightforward and efficient.

No external libraries were used for setting up the HTTP server; the standard library was sufficient for this use case.

This project also implements a Dead Letter Queue for handling messages that were not be able to be processed for some reason

This project is also fully dockerized with dependencies like RabbitMQ

This project currently consists of two files:

* `cmd/consumer/consumer.go`: This is an HTTP server that exposes a /notify endpoint.
When a request is made to this endpoint with an email and message, the server publishes the notification to the RabbitMQ queue.

* `cmd/producer/producer.go`: This is the consumer application.
It reads messages from the queue and uses multiple concurrent workers to process and send the notifications asynchronously.

## Usage

### Using docker

`docker-compose up --build`

### Manual

#### Copy the .env.test and change fields if necessary
`cp .env.test .env`

#### To start the consumer
`go run cmd/consumer/consumer.go`

#### To start the producer
`go run cmd/producer/producer.go`

#### Make sure the RabbitMQ instance is running, if not:
`docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:4-management`

## TODO
- [ ] Add retry logic if sending fails
- [ ] Add basic email format validation
- [ ] Write unit tests for sender and receiver
- [x] ~~Add Dockerfile to run locally with RabbitMQ easily (Dockerize entire application)~~
- [ ] Add integration testing
- [x] ~~Add Dead-Letter-Queue~~
