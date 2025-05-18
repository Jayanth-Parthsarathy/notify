package consumer

import (
	"errors"
	"testing"

	constants "github.com/jayanth-parthsarathy/notify/internal/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) Publish(exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error {
	args := m.Called(exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

type MockDelivery struct {
	mock.Mock
}

func (a *MockDelivery) Ack(multiple bool) error {
	args := a.Called(multiple)
	return args.Error(0)
}
func (a *MockDelivery) Nack(multiple, requeue bool) error {
	return a.Called(multiple, requeue).Error(0)
}
func (m *MockDelivery) Body() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *MockDelivery) ContentType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDelivery) Headers() amqp.Table {
	args := m.Called()
	return args.Get(0).(amqp.Table)
}

func TestRetry_MaxRetries(t *testing.T) {
	ch := new(MockChannel)
	msg := new(MockDelivery)

	body := []byte("payload")
	msg.On("Nack", false, false).Return(nil)
	msg.On("Body").Return(body)

	retry(ch, msg, 3)

	msg.AssertCalled(t, "Nack", false, false)
	ch.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	msg.AssertNotCalled(t, "Ack", mock.Anything)
}
func TestRetry_SingleRetry(t *testing.T) {
	ch := new(MockChannel)
	msg := new(MockDelivery)

	body := []byte("payload")
	contentType := "application/json"

	msg.On("Body").Return(body)
	msg.On("Headers").Return(amqp.Table{})
	msg.On("ContentType").Return(contentType)

	msg.On("Ack", false).Return(nil)

	ch.On("Publish",
		constants.RetryExchangeName,
		constants.Retry10sQueue,
		false, false,
		mock.MatchedBy(func(p amqp.Publishing) bool {
			got := getRetryCount(p.Headers)
			return got == 1 &&
				string(p.Body) == string(body) &&
				p.ContentType == contentType
		}),
	).Return(nil)

	retry(ch, msg, 1)

	msg.AssertCalled(t, "Ack", false)
	msg.AssertNotCalled(t, "Nack", mock.Anything)
	ch.AssertExpectations(t)
	msg.AssertExpectations(t)
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(recipient string, body string, subject string) error {
	args := m.Called(recipient, body, subject)
	return args.Error(0)
}

func TestProcessMessage_MalformedJSON(t *testing.T) {
	d := new(MockDelivery)
	ch := new(MockChannel)
	em := new(MockEmailSender)

	d.On("Body").Return([]byte("not-json"))
	d.On("Headers").Return(amqp.Table(nil))
	d.On("Nack", false, false).Return(nil)

	processMessage(d, ch, em)

	d.AssertCalled(t, "Nack", false, false)
	d.AssertNotCalled(t, "Ack", mock.Anything)
	ch.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	em.AssertNotCalled(t, "SendEmail", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_Success(t *testing.T) {
	d := new(MockDelivery)
	ch := new(MockChannel)
	em := new(MockEmailSender)

	valid := `{"email":"foo@bar.com","message":"hello", "subject": "hello world"}`
	d.On("Body").Return([]byte(valid))
	d.On("Headers").Return(amqp.Table(nil))
	d.On("Ack", false).Return(nil)
	em.On("SendEmail", "foo@bar.com", "hello", "hello world").Return(nil)

	processMessage(d, ch, em)

	d.AssertCalled(t, "Ack", false)
	d.AssertNotCalled(t, "Nack", mock.Anything, mock.Anything)
	ch.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	em.AssertCalled(t, "SendEmail", "foo@bar.com", "hello", "hello world")
}

func TestProcessMessage_FailureRetries(t *testing.T) {
	d := new(MockDelivery)
	ch := new(MockChannel)
	em := new(MockEmailSender)

	valid := `{"email":"foo@bar.com","message":"hello", "subject":"hello world"}`
	d.On("Body").Return([]byte(valid))
	d.On("Headers").Return(amqp.Table{"x-retry-count": int32(0)})
	d.On("ContentType").Return("application/json")
	d.On("Ack", false).Return(nil)
	ch.On("Publish",
		constants.RetryExchangeName,
		constants.Retry10sQueue,
		false, false,
		mock.MatchedBy(func(pub amqp.Publishing) bool {
			v := getRetryCount(pub.Headers)
			return v == 1
		}),
	).Return(nil)
	em.On("SendEmail", "foo@bar.com", "hello", "hello world").Return(errors.New("This is testing error"))

	processMessage(d, ch, em)

	ch.AssertCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	em.AssertCalled(t, "SendEmail", "foo@bar.com", "hello", "hello world")
}

// should not retry
func TestProcessMessage_FailureInvalidEmail(t *testing.T) {
	d := new(MockDelivery)
	ch := new(MockChannel)
	em := new(MockEmailSender)

	// invalid email
	valid := `{"email":"foo","message":"hello", "subject":"hello world"}`
	d.On("Body").Return([]byte(valid))
	d.On("Headers").Return(amqp.Table{"x-retry-count": int32(0)})
	d.On("ContentType").Return("application/json")
	d.On("Ack", false).Return(nil)
	d.On("Nack", false, false).Return(nil)
	ch.On("Publish",
		constants.RetryExchangeName,
		constants.Retry10sQueue,
		false, false,
		mock.MatchedBy(func(pub amqp.Publishing) bool {
			v := getRetryCount(pub.Headers)
			return v == 1
		}),
	).Return(nil)
	em.On("SendEmail", "foo", "hello", "hello world").Return(&InvalidEmailError{email: "foo", message: "Invalid email sending it to DLQ"})

	processMessage(d, ch, em)

	ch.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	em.AssertCalled(t, "SendEmail", "foo", "hello", "hello world")
	d.AssertCalled(t, "Nack", false, false)
}
