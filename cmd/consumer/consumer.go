package main

import (
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/consumer"
	consumer_types "github.com/jayanth-parthsarathy/notify/internal/consumer/types"
)

func main() {
	util.LoadEnv()
	conn := util.ConnectToRabbitMQ()
	defer conn.Close()
	util.DeclareQueue(conn)
	gmailSender := consumer_types.GmailSender{}
	consumer.StartWorkers(conn, &gmailSender)
}
