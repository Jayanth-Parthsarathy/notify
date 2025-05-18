package common

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"testing"
)

func TestFailOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic but did not get one")
		}
	}()
	err := fmt.Errorf("This is a error")
	FailOnError(err, "This should panic")
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	err := fmt.Errorf("This is a error")
	LogError(err, "ABCD")
	expectedPattern := `^.*ABCD : This is a error`
	matched, matchErr := regexp.MatchString(expectedPattern, buf.String())
	if matchErr != nil {
		t.Errorf("Error in regex match: %s", err)
	}
	if !matched {
		t.Errorf("Log output did not match expected pattern. Got: %s", buf.String())
	}
}
