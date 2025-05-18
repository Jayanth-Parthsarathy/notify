package integration

// import (
// 	"os"
// 	"testing"
//
// 	"github.com/jayanth-parthsarathy/notify/internal/common/util"
// )

// func TestRabbitMQConnection(t *testing.T) {
// 	os.Setenv("RABBIT_MQ_URL", "amqp://guest:guest@localhost:5672/")
//
// 	conn, ch := util.ConnectToRabbitMQ()
// 	defer conn.Close()
// 	defer ch.Close()
//
// 	if conn.IsClosed() {
// 		t.Fatal("Expected connection to be open")
// 	}
//
// 	if ch == nil {
// 		t.Fatal("Expected channel to be non-nil")
// 	}
// }
//
// func TestRabbitMQDeclareQueue(t *testing.T) {
// 	os.Setenv("RABBIT_MQ_URL", "amqp://guest:guest@localhost:5672/")
//
// 	conn, ch := util.ConnectToRabbitMQ()
// 	defer conn.Close()
// 	defer ch.Close()
//
// 	q, _, _ := util.DeclareQueue(ch)
//
// 	if q == nil || q.Name != "notification_queue" {
// 		t.Fatalf("Expected queue 'notification_queue', got: %v", q)
// 	}
// }
