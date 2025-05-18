package main

import (
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/producer"
)


func main() {
	util.LoadEnv()
	conn, ch := util.ConnectToRabbitMQ()
	defer conn.Close()
	defer ch.Close()
	q := util.DeclareQueue(ch)
	producer.StartServer(q, ch)
}
