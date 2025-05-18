package common

import (
	"fmt"
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
