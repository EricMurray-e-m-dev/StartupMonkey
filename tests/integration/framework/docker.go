package framework

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (e *TestEnvironment) Start() error {
	e.t.Logf("Starting Docker Services: %v", e.Services)

	buildCmd := exec.Command("docker-compose",
		"-f", e.ComposeFile,
		"-p", e.ProjectName,
		"build",
	)

	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker-compose build failed: %w\n%s", err, output)
	}

	args := []string{"-f", e.ComposeFile, "-p", e.ProjectName, "up", "-d"}
	args = append(args, e.Services...)

	cmd := exec.Command("docker-compose", args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker-compose build failed: %w\n%s", err, output)
	}

	e.t.Log("Docker services started")
	return nil
}

func (e *TestEnvironment) WaitForHealthy(timeout time.Duration) error {
	e.t.Logf("Waiting for services to be healthy (timeout: %v)", timeout)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.Command("docker-compose",
			"-f", e.ComposeFile,
			"-p", e.ProjectName,
			"ps", "--services", "--filter", "status=running",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to check service status: %v", err)
		}

		runningServices := strings.Split(strings.TrimSpace(string(output)), "\n")

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
			e.t.Logf("All services healthy")
			return nil
		}

		time.Sleep(2 * time.Second)

	}

	return fmt.Errorf("services health check timed out")
}

func (e *TestEnvironment) GetLogs(serviceName string) (string, error) {
	cmd := exec.Command("docker-compose",
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

func (e *TestEnvironment) Cleanup() {
	e.t.Logf("Cleaning up docker services...")

	cmd := exec.Command("docker-compose",
		"-f", e.ComposeFile,
		"-p", e.ProjectName,
		"down", "-v",
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Logf("Warning: cleanup failed %v\n%s", err, output)
	}

	duration := time.Since(e.StartTime)
	e.t.Logf("Test completed in: %v", duration)
}
