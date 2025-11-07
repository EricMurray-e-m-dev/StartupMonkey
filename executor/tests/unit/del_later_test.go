package unit

import (
	"testing"
)

func Test_Skip(t *testing.T) {
	t.Skip("DetectionHandler testing to move to integration testing due to DB connection requirement")
}
