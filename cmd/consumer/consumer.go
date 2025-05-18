package main

import (
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/consumer"
	consumer_types "github.com/jayanth-parthsarathy/notify/internal/consumer/types"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		PadLevelText:    true,
	})
	util.LoadEnv()
	conn := util.ConnectToRabbitMQ()
	defer conn.Close()
	util.DeclareQueue(conn)
	gmailSender := consumer_types.GmailSender{}
	consumer.StartWorkers(conn, &gmailSender)
}
