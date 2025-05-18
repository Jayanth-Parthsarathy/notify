package producer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jayanth-parthsarathy/notify/internal/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateRequestBody_ValidJSON(t *testing.T) {
	validJSON := `{"email":"test@example.com","message":"Hello", "subject":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(validJSON))
	w := httptest.NewRecorder()

	body := validateRequestBody(w, req)
	require.NotNil(t, body)
	require.JSONEq(t, validJSON, string(body))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestValidateRequestBody_InvalidJSON(t *testing.T) {
	invalidJSON := `{"email":"test@example.com", "message": }`
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(invalidJSON))
	w := httptest.NewRecorder()

	body := validateRequestBody(w, req)
	require.Nil(t, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Invalid request body")
}

func TestWriteSuccessResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonBody := []byte(`{"foo":"bar"}`)

	writeSuccessResponse(rr, jsonBody)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("Status = %d; want %d", got, want)
	}

	if got, want := strings.TrimSpace(rr.Body.String()), "Notification queued successfully"; got != want {
		t.Errorf("Body = %q; want %q", got, want)
	}
}

type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) PublishWithContext(ctx context.Context, exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error {
	args := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

func (m *MockChannel) Close() error {
	return m.Called().Error(0)
}

func TestPublishMessage_Success(t *testing.T) {
	mockCh := new(MockChannel)
	ctx := context.Background()
	body := []byte(`{"msg":"hello"}`)
	recorder := httptest.NewRecorder()

	mockCh.On("PublishWithContext", ctx, "", constants.MainQueueName, false, false, mock.MatchedBy(func(p amqp.Publishing) bool {
		return p.ContentType == "application/json" && bytes.Equal(p.Body, body)
	})).Return(nil)

	err := publishMessage(body, mockCh, recorder, ctx)

	assert.NoError(t, err)
	mockCh.AssertExpectations(t)
}

func TestPublishMessage_Failure(t *testing.T) {
	mockCh := new(MockChannel)
	ctx := context.Background()
	body := []byte(`{"msg":"fail"}`)
	recorder := httptest.NewRecorder()

	mockCh.On("PublishWithContext", ctx, "", constants.MainQueueName, false, false, mock.MatchedBy(func(p amqp.Publishing) bool {
		return p.ContentType == "application/json" && bytes.Equal(p.Body, body)
	})).Return(assert.AnError)

	err := publishMessage(body, mockCh, recorder, ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Result().StatusCode)
	mockCh.AssertExpectations(t)
}

func TestHandleNotification(t *testing.T) {
	const uri = "/notify"

	type testCase struct {
		name           string
		method         string
		body           string
		setupMock      func(m *MockChannel)
		wantCode       int
		wantBodySubstr string
	}

	cases := []testCase{
		{
			name:           "non-POST gives 405",
			method:         http.MethodGet,
			body:           "",
			wantCode:       http.StatusMethodNotAllowed,
			wantBodySubstr: "Only post method is accepted",
		},
		{
			name:           "invalid JSON gives 400",
			method:         http.MethodPost,
			body:           `{"foo":}`,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "Invalid request body",
		},
		{
			name:   "publish error gives 500",
			method: http.MethodPost,
			body:   `{"email":"x@x","message":"y"}`,
			setupMock: func(m *MockChannel) {
				m.On("Close").Return(nil)
				m.On("PublishWithContext",
					mock.Anything, "", constants.MainQueueName, false, false, mock.Anything,
				).Return(errors.New("boom"))
			},
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: "could not queue notification",
		},
		{
			name:   "happy path gives 200",
			method: http.MethodPost,
			body:   `{"email":"a@b","message":"ok"}`,
			setupMock: func(m *MockChannel) {
				m.On("Close").Return(nil)
				m.On("PublishWithContext",
					mock.Anything,
					"", constants.MainQueueName, false, false,
					mock.MatchedBy(func(p amqp.Publishing) bool {
						return p.ContentType == "application/json"
					}),
				).Return(nil)
			},
			wantCode:       http.StatusOK,
			wantBodySubstr: "Notification queued successfully",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCh := new(MockChannel)

			if tc.setupMock != nil {
				tc.setupMock(mockCh)
			} else {
				mockCh.On("Close").Return(nil)
			}

			req := httptest.NewRequest(tc.method, uri, strings.NewReader(tc.body))
			rr := httptest.NewRecorder()

			handleNotification(rr, req, mockCh)
			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tc.wantCode, res.StatusCode)
			bodyBytes, _ := io.ReadAll(res.Body)
			assert.Contains(t, string(bodyBytes), tc.wantBodySubstr)

			mockCh.AssertExpectations(t)
		})
	}
}
