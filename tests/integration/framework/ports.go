package framework

import (
	"os/exec"
	"strings"
)

func (e *TestEnvironment) GetPublishedPort(service string, containerPort string) (string, error) {
	cmd := exec.Command("docker", "compose",
		"-f", e.ComposeFile,
		"-p", e.ProjectName,
		"port", service, containerPort)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	return parts[len(parts)-1], nil
}
