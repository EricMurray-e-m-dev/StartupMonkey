package framework

import (
	"testing"
	"time"
)

type TestEnvironment struct {
	t           *testing.T
	ComposeFile string
	ProjectName string
	Services    []string
	StartTime   time.Time
}

func NewTestEnvironment(t *testing.T, services []string) *TestEnvironment {
	return &TestEnvironment{
		t:           t,
		ComposeFile: "../../docker-compose.yml",
		ProjectName: "startupmonkey-test",
		Services:    services,
		StartTime:   time.Now(),
	}
}
