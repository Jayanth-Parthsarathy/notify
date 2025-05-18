package common

type RequestBody struct {
	Email   string `json:"email"`
	Message string `json:"message"`
	Subject string `json:"subject"`
}
