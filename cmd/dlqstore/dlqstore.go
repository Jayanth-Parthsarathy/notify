package main

import (
	"github.com/jayanth-parthsarathy/notify/internal/common/constants"
	"github.com/jayanth-parthsarathy/notify/internal/common/util"
	"github.com/jayanth-parthsarathy/notify/internal/dlqstore"
	"github.com/jayanth-parthsarathy/notify/internal/dlqstore/types"
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
	conn := util.ConnectToRabbitMQ()
	var connAdapter = dlqstore_types.NewConnectionAdapter(conn)
	db := util.ConnectToDB()
	inspector := dlqstore.NewPgInspector(db, connAdapter, constants.MainQueueName)
	dlqstore.StartServer(conn, inspector, db)
}
