# Notify

This is a simple notification service built using Golang that uses RabbitMQ as a queue, with asynchronous workers to poll and send messages.

RabbitMQ is used to improve the scalability of the system.
Instead of overloading the main server with message processing, we offload messages to a queue, where they are handled asynchronously by worker processes.

I have used Golang for this project because it is simple to use and easy to get started with.
Additionally, goroutines and channels in Go make implementing concurrency straightforward and efficient.

No external libraries were used for setting up the HTTP server; the standard library was sufficient for this use case.

This project currently consists of two files:

* sender.go: This is an HTTP server that exposes a /notify endpoint.
When a request is made to this endpoint with an email and message, the server publishes the notification to the RabbitMQ queue.

* receiver/receiver.go: This is the consumer application.
It reads messages from the queue and uses multiple concurrent workers to process and send the notifications asynchronously.

## TODO
- Add retry logic if sending fails
- Add basic email format validation
- Save failed messages to database for later retry
- Write unit tests for sender and receiver
- Add Dockerfile to run locally with RabbitMQ easily
