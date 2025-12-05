package actions

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/docker"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	dockertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// DeployRedisAction deploys a Redis container for application-level caching.
// This is an ADVANCED action that requires the user to modify their application code.
type DeployRedisAction struct {
	actionID      string
	detectionID   string
	databaseID    string
	containerName string
	containerID   string
	deployed      bool

	// Docker client
	dockerClient *docker.Client

	// Redis configuration
	port           string
	maxMemory      string
	evictionPolicy string

	// Store deployment details for rollback
	deploymentDetails map[string]interface{}
}

// NewDeployRedisAction creates a new Redis deployment action.
func NewDeployRedisAction(
	actionID string,
	detectionID string,
	databaseID string,
	params map[string]interface{},
) (*DeployRedisAction, error) {
	containerName := fmt.Sprintf("redis-%s", databaseID)

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Parse configuration from params (with defaults)
	port := "6379"
	maxMemory := "256mb"
	evictionPolicy := "allkeys-lru"

	if p, ok := params["port"].(string); ok && p != "" {
		port = p
	}
	if m, ok := params["max_memory"].(string); ok && m != "" {
		maxMemory = m
	}
	if e, ok := params["eviction_policy"].(string); ok && e != "" {
		evictionPolicy = e
	}

	return &DeployRedisAction{
		actionID:          actionID,
		detectionID:       detectionID,
		databaseID:        databaseID,
		containerName:     containerName,
		dockerClient:      dockerClient,
		port:              port,
		maxMemory:         maxMemory,
		evictionPolicy:    evictionPolicy,
		deployed:          false,
		deploymentDetails: make(map[string]interface{}),
	}, nil
}

func (a *DeployRedisAction) Execute(ctx context.Context) (*models.ActionResult, error) {
	startTime := time.Now()

	if a.actionID == "" {
		return nil, fmt.Errorf("action ID not set")
	}

	var containerID string
	var message string

	// Check if container already exists
	exists, existingID, err := a.dockerClient.ContainerExists(ctx, a.containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if container exists: %w", err)
	}

	if exists {
		// Container exists - check if it's running
		isRunning, err := a.dockerClient.IsContainerRunning(ctx, existingID)
		if err != nil {
			return nil, fmt.Errorf("failed to check container status: %w", err)
		}

		if isRunning {
			log.Printf("Redis is already running on port %s", a.port)

			endTime := time.Now()
			return &models.ActionResult{
				ActionID:        a.actionID,
				DetectionID:     a.detectionID,
				ActionType:      "deploy_redis",
				DatabaseID:      a.databaseID,
				Status:          models.StatusCompleted,
				Message:         "Redis is already running (no action needed)",
				CreatedAt:       startTime,
				Started:         &startTime,
				Completed:       &endTime,
				ExecutionTimeMs: endTime.Sub(startTime).Milliseconds(),
				Changes: map[string]interface{}{
					"container_name":    a.containerName,
					"container_id":      existingID,
					"redis_port":        a.port,
					"instruction":       "Redis already deployed",
					"connection_string": fmt.Sprintf("redis://localhost:%s", a.port),
				},
				CanRollback: true,
				Rolledback:  false,
			}, nil
		}

		// Container exists but is stopped - restart it
		log.Printf("Restarting existing Redis container: %s", existingID[:12])

		if err := a.dockerClient.StartContainer(ctx, existingID); err != nil {
			return nil, fmt.Errorf("failed to restart container: %w", err)
		}

		containerID = existingID
		a.containerID = existingID
		message = fmt.Sprintf("Redis container '%s' restarted on port %s", a.containerName, a.port)

	} else {
		// Container doesn't exist - create new one
		log.Printf("Deploying new Redis container...")

		// Pull Redis image
		log.Printf("Pulling Redis image...")
		if err := a.dockerClient.PullImage(ctx, "redis:7-alpine"); err != nil {
			return nil, fmt.Errorf("failed to pull Redis image: %w", err)
		}

		// Create container
		log.Printf("Creating Redis container: %s", a.containerName)

		portBindings := nat.PortMap{
			"6379/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: a.port,
				},
			},
		}

		// Redis configuration via command arguments
		cmd := []string{
			"redis-server",
			"--maxmemory", a.maxMemory,
			"--maxmemory-policy", a.evictionPolicy,
			"--appendonly", "yes", // Enable persistence
		}

		containerConfig := &dockertypes.Config{
			Image: "redis:7-alpine",
			Cmd:   cmd,
			ExposedPorts: nat.PortSet{
				"6379/tcp": struct{}{},
			},
		}

		hostConfig := &dockertypes.HostConfig{
			PortBindings: portBindings,
			RestartPolicy: dockertypes.RestartPolicy{
				Name: "unless-stopped",
			},
		}

		newContainerID, err := a.dockerClient.CreateContainer(ctx, containerConfig, hostConfig, a.containerName)
		if err != nil {
			return nil, fmt.Errorf("failed to create container: %w", err)
		}

		containerID = newContainerID
		a.containerID = containerID
		log.Printf("Container created: %s", containerID[:12])

		// Start container
		log.Printf("Starting Redis container...")
		if err := a.dockerClient.StartContainer(ctx, containerID); err != nil {
			a.dockerClient.RemoveContainer(ctx, containerID)
			return nil, fmt.Errorf("failed to start container: %w", err)
		}

		message = fmt.Sprintf("Redis deployed as container '%s' on port %s", a.containerName, a.port)
	}

	// Verify running
	log.Printf("Verifying Redis is running...")
	time.Sleep(2 * time.Second)

	isRunning, err := a.dockerClient.IsContainerRunning(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check container status: %w", err)
	}

	if !isRunning {
		return nil, fmt.Errorf("container started but is not running - check logs with: docker logs %s", a.containerName)
	}

	log.Printf("Redis is running on port %s", a.port)

	endTime := time.Now()
	executionTimeMs := endTime.Sub(startTime).Milliseconds()

	result := &models.ActionResult{
		ActionID:        a.actionID,
		DetectionID:     a.detectionID,
		ActionType:      "deploy_redis",
		DatabaseID:      a.databaseID,
		Status:          models.StatusCompleted,
		Message:         message,
		CreatedAt:       startTime,
		Started:         &startTime,
		Completed:       &endTime,
		ExecutionTimeMs: executionTimeMs,
		Changes: map[string]interface{}{
			"container_name":       a.containerName,
			"container_id":         containerID,
			"redis_port":           a.port,
			"max_memory":           a.maxMemory,
			"eviction_policy":      a.evictionPolicy,
			"connection_string":    fmt.Sprintf("redis://localhost:%s", a.port),
			"instruction":          "Update your application to use Redis for caching. See integration guide in Dashboard.",
			"requires_code_change": true,
		},
		CanRollback: true,
		Rolledback:  false,
	}

	a.deployed = true
	return result, nil
}

func (a *DeployRedisAction) Rollback(ctx context.Context) error {
	if !a.deployed {
		return fmt.Errorf("Redis was not deployed, cannot rollback")
	}

	if a.containerID == "" {
		return fmt.Errorf("container ID not found")
	}

	// Stop the container
	if err := a.dockerClient.StopContainer(ctx, a.containerID); err != nil {
		log.Printf("Warning: failed to stop container: %v", err)
	}

	// Remove the container
	if err := a.dockerClient.RemoveContainer(ctx, a.containerID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("Redis container removed: %s", a.containerName)

	a.deployed = false
	return nil
}

func (a *DeployRedisAction) Validate(ctx context.Context) error {
	// Check Docker is available
	if err := a.dockerClient.IsAvailable(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check port is not already in use (by non-Redis container)
	// This is a basic check - full port scanning would be more robust
	// but Docker will fail to bind if port is in use anyway

	return nil
}

func (a *DeployRedisAction) GetMetadata() *models.ActionMetadata {
	return &models.ActionMetadata{
		ActionID:     a.actionID,
		ActionType:   "deploy_redis",
		DatabaseID:   a.databaseID,
		DatabaseType: "redis",
		CreatedAt:    time.Now(),
	}
}
