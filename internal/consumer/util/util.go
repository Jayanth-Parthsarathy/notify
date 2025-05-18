package consumer_util

import "net/mail"

func Valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

