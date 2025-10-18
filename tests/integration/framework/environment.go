package framework

import (
	"fmt"
	"testing"
	"time"
)

type TestEnvironment struct {
	t               *testing.T
	ComposeFile     string
	ComposeTestFile string
	ProjectName     string
	Services        []string
	StartTime       time.Time
}

func NewTestEnvironment(t *testing.T, services []string) *TestEnvironment {
	projectName := fmt.Sprintf("startupmonkey-test-%d", time.Now().Unix())

	return &TestEnvironment{
		t:               t,
		ComposeFile:     "../../docker-compose.yml",
		ComposeTestFile: "../../docker-compose.test.yml",
		ProjectName:     projectName,
		Services:        services,
		StartTime:       time.Now(),
	}
}
