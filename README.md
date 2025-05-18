# Notify

**Notify** is a simple notification service built in Go that uses RabbitMQ for asynchronous message processing and retries.

It is designed to offload message handling from the main server to a queue, where concurrent workers process notifications in the background. This improves scalability and fault tolerance.

## ✨ Features

* Built using Golang with standard library

* Uses goroutines for concurrent execution of sending notifications

* Fully Dockerized with RabbitMQ services included

* Implements exponential backoff retry queues (10s → 30s → 60s)

* Dead-Letter-Queue for handling messages that have exceeded max-retries and failed (Logs to a file)

* Unit tests with mock (via Testify) for most of the internal logic

* Sends emails using `net/smtp` via an abstracted `EmailSender` interface

* Includes load tests via `k6` in `load_test.js`

It currently consists of two main files:

* `cmd/consumer/consumer.go`: This is an HTTP server that exposes a /notify endpoint.
When a request is made to this endpoint with an email and message, the server publishes the notification to the RabbitMQ queue.

* `cmd/producer/producer.go`: This is the consumer application.
It reads messages from the queue and uses multiple concurrent workers to process and send the notifications asynchronously.

## 🚀 Usage

### 📦 Docker (Recommended)

```bash
docker-compose --env-file .env.docker up --build
```

### 🛠️ Manual

#### Copy the .env.test and add the missing fields
```bash
cp .env.test .env
```

#### To start the consumer
```bash
go run cmd/consumer/consumer.go
```

#### To start the producer
```bash
go run cmd/producer/producer.go
```

#### Make sure the RabbitMQ instance is running, if not:
```bash
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:4-management
```
### 🧪 Testing
```bash
go test ./...
```
All core components are covered by unit tests with mock logic for external dependencies.

### 📬 API Endpoint
`POST /notify`

```json
{
  "email": "user@example.com",
  "message": "Hello from Notify!",
  "subject": "Hello"
}
```

## TODO
- [x] ~~Add retry logic if sending fails~~
- [x] ~~Write unit tests for sender and receiver~~
- [x] ~~Add Dockerfile to run locally with RabbitMQ easily (Dockerize entire application)~~
- [x] ~~Add Dead-Letter-Queue~~
- [x] ~~Add basic email format validation~~
- [ ] Add integration testing
- [ ] Create system architecture diagram for understanding
