package main

import (
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/consumer"
)

func main() {
	util.LoadEnv()
	conn := util.ConnectToRabbitMQ()
	defer conn.Close()
	util.DeclareQueue(conn)
	consumer.StartWorkers(conn)
}
