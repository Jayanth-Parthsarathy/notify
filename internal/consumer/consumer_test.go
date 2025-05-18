package consumer

import (
	"testing"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestGetRetryCount(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(0, getRetryCount(nil))
	assert.Equal(0, getRetryCount(amqp.Table{}))
	assert.Equal(2, getRetryCount(amqp.Table{"x-retry-count": int32(2)}))
	assert.Equal(2, getRetryCount(amqp.Table{"x-retry-count": int64(2)}))
	assert.Equal(3, getRetryCount(amqp.Table{"x-retry-count": 3}))
	assert.Equal(0, getRetryCount(amqp.Table{"x-retry-count": "bad"}))
}

func TestGetRetryQueueName(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(constants.Retry10sQueue, getRetryQueueName(1), "Should return 10s retry queue")
	assert.Equal(constants.Retry30sQueue, getRetryQueueName(2), "Should return 30s retry queue")
	assert.Equal(constants.Retry60sQueue, getRetryQueueName(3), "Should return 60s retry queue")
	assert.Equal("", getRetryQueueName(4), "Should return empty string for invalid retry count")
}

func TestPopulateHeader(t *testing.T) {
	assert := assert.New(t)

	h := populateHeader(nil, 2)
	assert.NotNil(h)
	assert.Equal(int32(2), h["x-retry-count"])

	headers := amqp.Table{
		"some-key": "some-value",
	}
	h2 := populateHeader(headers, 3)
	assert.Equal("some-value", h2["some-key"])
	assert.Equal(int32(3), h2["x-retry-count"])

	headers = amqp.Table{
		"x-retry-count": "0",
	}
	h3 := populateHeader(headers, 3)
	assert.Equal(int32(3), h3["x-retry-count"])
}
