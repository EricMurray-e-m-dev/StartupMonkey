package framework

import (
	"strings"
	"testing"
)

// AssertLogsContain checks that service logs contain expected string
func AssertLogsContain(t *testing.T, env *TestEnvironment, serviceName string, expected string) {
	t.Helper()

	logs, err := env.GetLogs(serviceName)
	if err != nil {
		t.Fatalf("Failed to get logs for %s: %v", serviceName, err)
	}

	if !strings.Contains(logs, expected) {
		t.Errorf("Expected logs to contain '%s', but it was not found in:\n%s", expected, logs)
	}
}

// AssertServiceRunning checks that a service is running
func AssertServiceRunning(t *testing.T, env *TestEnvironment, serviceName string) {
	t.Helper()

	// Service should be in the running services list if healthy
	// This is implicitly checked by WaitForHealthy, but we can add explicit check
	logs, err := env.GetLogs(serviceName)
	if err != nil {
		t.Fatalf("Service %s is not running: %v", serviceName, err)
	}

	if logs == "" {
		t.Errorf("Service %s has no logs (may not be running)", serviceName)
	}
}
