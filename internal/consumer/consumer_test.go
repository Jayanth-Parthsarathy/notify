package consumer

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestProcessMessage(t *testing.T) {
}

func TestWorker(t *testing.T) {
}

func TestGetRetryCount(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{"nil headers", nil, 0},
		{"missing key", amqp.Table{}, 0},
		{"int32", amqp.Table{"x-retry-count": int32(2)}, 2},
		{"int", amqp.Table{"x-retry-count": 3}, 3},
		{"bad type", amqp.Table{"x-retry-count": "foo"}, 0},
	}
	for _, tt := range tests {
		got := getRetryCount(tt.headers)
		if got != tt.want {
			t.Errorf("getRetryCount(%v) = %d, want %d", tt.headers, got, tt.want)
		}
	}
}
