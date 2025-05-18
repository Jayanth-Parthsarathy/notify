package dlqstore

import (
	"context"
	"encoding/json"
	"testing"

	dlqstore_types "github.com/jayanth-parthsarathy/notify/internal/dlqstore/types"
	"github.com/pashagolub/pgxmock"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) Get(queue string, autoAck bool) (amqp.Delivery, bool, error) {
	args := m.Called(queue, autoAck)
	return args.Get(0).(amqp.Delivery), args.Bool(1), args.Error(2)
}

func (m *MockChannel) Nack(tag uint64, multiple, requeue bool) error {
	args := m.Called(tag, multiple, requeue)
	return args.Error(0)
}

func (m *MockChannel) Ack(tag uint64, multiple bool) error {
	args := m.Called(tag, multiple)
	return args.Error(0)
}

func (m *MockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

func (m *MockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	args := m.Called(prefetchCount, prefetchSize, global)
	return args.Error(0)
}

func (m *MockChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockConnection struct {
	mock.Mock
	ChannelMock *MockChannel
}

func (m *MockConnection) Channel() (dlqstore_types.AMQPChannel, error) {
	m.Called()
	return m.ChannelMock, nil
}

func TestPostgresInspector_ListMessages(t *testing.T) {
	mockDB, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer mockDB.Close(context.Background())

	headers := amqp.Table{"foo": "bar"}
	hdrBytes, _ := json.Marshal(headers)
	bodyBytes := []byte(`{"hello":"world"}`)

	rows := pgxmock.NewRows([]string{"message_id", "headers", "body"}).
		AddRow("msg-1", hdrBytes, bodyBytes)

	mockDB.ExpectQuery(`SELECT message_id, headers, body FROM dlq_messages ORDER BY received_at DESC LIMIT \$1`).
		WithArgs(5).
		WillReturnRows(rows)

	inspector := NewPgInspector(mockDB, nil /* conn */, "unused")

	msgs, err := inspector.ListMessages(5)
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "msg-1", msgs[0].ID)
	assert.Equal(t, `{"hello":"world"}`, msgs[0].Payload)
	assert.Equal(t, headers, msgs[0].Headers)

	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestPostgresInspector_RequeueMessage(t *testing.T) {
	mockDB, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer mockDB.Close(context.Background())

	headers := amqp.Table{"foo": "baz"}
	hdrBytes, _ := json.Marshal(headers)
	bodyBytes := []byte(`{"ping":"pong"}`)

	mockDB.ExpectQuery(`SELECT headers, body FROM dlq_messages WHERE message_id = \$1`).
		WithArgs("the-id").
		WillReturnRows(
			pgxmock.NewRows([]string{"headers", "body"}).
				AddRow(hdrBytes, bodyBytes),
		)

	mockDB.ExpectExec(`DELETE FROM dlq_messages WHERE message_id = \$1`).
		WithArgs("the-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mockCh := &MockChannel{}
	mockCh.On("Publish", "", "retry-queue", false, false, mock.MatchedBy(func(pub amqp.Publishing) bool {
		return string(pub.Body) == string(bodyBytes) &&
			pub.MessageId == "the-id"
	})).Return(nil)
	mockCh.On("Close").Return(nil)

	mockConn := &MockConnection{ChannelMock: mockCh}
	mockConn.On("Channel").Return(mockCh, nil)

	inspector := NewPgInspector(mockDB, mockConn, "retry-queue")

	err = inspector.RequeueMessage("the-id")
	assert.NoError(t, err)

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockCh.AssertExpectations(t)
}
