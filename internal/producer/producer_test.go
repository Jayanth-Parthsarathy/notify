package producer

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRequestBody_ValidJSON(t *testing.T) {
	validJSON := `{"email":"test@example.com","message":"Hello"}`
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
