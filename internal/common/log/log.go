package common

import (
	log "github.com/sirupsen/logrus"
)

func FailOnError(err error, msg string) {
	if err != nil {
		log.Errorf("%s : %s", msg, err)
		panic(err)
	}
}

func LogError(err error, msg string) {
	if err != nil {
		log.Errorf("%s : %s", msg, err)
	}
}
