package framework

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (e *TestEnvironment) Start() error {
	e.t.Logf("Starting Docker Services: %v", e.Services)

	// Build images first
	buildCmd := exec.Command("docker", "compose",
		"-f", e.ComposeFile,
		"-f", e.ComposeTestFile,
		"-p", e.ProjectName,
		"build",
	)

	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker-compose build failed: %w\n%s", err, output)
	}

	// Start services
	args := []string{"compose", "-f", e.ComposeFile, "-f", e.ComposeTestFile, "-p", e.ProjectName, "up", "-d"}
	args = append(args, e.Services...)

	cmd := exec.Command("docker", args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker-compose up failed: %w\n%s", err, output)
	}

	e.t.Log("Docker services started")
	return nil
}

// WaitForHealthy waits for all services to be healthy
func (e *TestEnvironment) WaitForHealthy(timeout time.Duration) error {
	e.t.Logf("Waiting for services to be healthy (timeout: %v)", timeout)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if all services are running
		cmd := exec.Command("docker", "compose",
			"-f", e.ComposeFile,
			"-p", e.ProjectName,
			"ps", "--services", "--filter", "status=running",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to check service status: %w", err)
		}

		runningServices := strings.Split(strings.TrimSpace(string(output)), "\n")

		// Check if all requested services are running
		allRunning := true
		for _, service := range e.Services {
			found := false
			for _, running := range runningServices {
				if running == service {
					found = true
					break
				}
			}
			if !found {
				allRunning = false
				break
			}
		}

		if allRunning {
			e.t.Log("All services healthy")
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("services did not become healthy within timeout")
}

// GetLogs retrieves logs from a specific service
func (e *TestEnvironment) GetLogs(serviceName string) (string, error) {
	cmd := exec.Command("docker", "compose",
		"-f", e.ComposeFile,
		"-p", e.ProjectName,
		"logs", serviceName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

// Cleanup stops and removes all containers
func (e *TestEnvironment) Cleanup() {
	e.t.Log("Cleaning up docker services...")

	cmd := exec.Command("docker", "compose",
		"-f", e.ComposeFile,
		"-p", e.ProjectName,
		"down", "-v",
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Logf("Warning: cleanup failed: %v\n%s", err, output)
	}

	duration := time.Since(e.StartTime)
	e.t.Logf("Test completed in: %v", duration)
}
