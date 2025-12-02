package actions

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/docker"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	dockertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type DeployPgBouncerAction struct {
	actionID      string
	detectionID   string
	databaseID    string
	databaseType  string
	containerName string
	containerID   string
	deployed      bool

	// Docker client
	dockerClient *docker.Client

	// Knowledge client for fetching connection string
	knowledgeClient pb.KnowledgeServiceClient

	// Store deployment details for rollback
	deploymentDetails map[string]interface{}
}

func NewDeployPgBouncerAction(actionID string, detectionID, databaseID, databaseType string, knowledgeClient pb.KnowledgeServiceClient, params map[string]interface{}) (*DeployPgBouncerAction, error) {
	containerName := fmt.Sprintf("pgbouncer-%s", databaseID)

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DeployPgBouncerAction{
		actionID:          actionID,
		detectionID:       detectionID,
		databaseID:        databaseID,
		databaseType:      databaseType,
		containerName:     containerName,
		dockerClient:      dockerClient,
		knowledgeClient:   knowledgeClient,
		deployed:          false,
		deploymentDetails: make(map[string]interface{}),
	}, nil
}

func (a *DeployPgBouncerAction) Execute(ctx context.Context) (*models.ActionResult, error) {
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
			log.Printf("PgBouncer is already running on port 6432")

			endTime := time.Now()
			return &models.ActionResult{
				ActionID:        a.actionID,
				DetectionID:     a.detectionID,
				ActionType:      "deploy_pgbouncer",
				DatabaseID:      a.databaseID,
				Status:          models.StatusCompleted,
				Message:         "PgBouncer is already running (no action needed)",
				CreatedAt:       startTime,
				Started:         &startTime,
				Completed:       &endTime,
				ExecutionTimeMs: endTime.Sub(startTime).Milliseconds(),
				Changes: map[string]interface{}{
					"container_name": a.containerName,
					"container_id":   existingID,
					"pgbouncer_port": 6432,
					"instruction":    "PgBouncer already deployed",
				},
				CanRollback: true,
				Rolledback:  false,
			}, nil
		}

		// Container exists but is stopped - restart it
		log.Printf("Restarting existing PgBouncer container: %s", existingID[:12])

		if err := a.dockerClient.StartContainer(ctx, existingID); err != nil {
			return nil, fmt.Errorf("failed to restart container: %w", err)
		}

		containerID = existingID
		a.containerID = existingID
		message = fmt.Sprintf("PgBouncer container '%s' restarted on port 6432", a.containerName)

	} else {
		// Container doesn't exist - create new one
		log.Printf("Deploying new PgBouncer container...")

		// Fetch database connection string from Knowledge
		log.Printf("Fetching database connection info from Knowledge...")
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dbCancel()

		dbResp, err := a.knowledgeClient.GetDatabase(dbCtx, &pb.GetDatabaseRequest{
			DatabaseId: a.databaseID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch database from Knowledge: %w", err)
		}

		if !dbResp.Found {
			return nil, fmt.Errorf("database not found in Knowledge: %s", a.databaseID)
		}

		connStr := dbResp.ConnectionString
		log.Printf("Retrieved connection string from Knowledge for database: %s", a.databaseID)

		// Parse connection string
		connStr = strings.TrimPrefix(connStr, "postgresql://")
		connStr = strings.TrimPrefix(connStr, "postgres://")

		if idx := strings.Index(connStr, "?"); idx != -1 {
			connStr = connStr[:idx]
		}

		parts := strings.Split(connStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid connection string format")
		}

		userPass := strings.Split(parts[0], ":")
		user := userPass[0]
		password := ""
		if len(userPass) == 2 {
			password = userPass[1]
		}

		hostDB := strings.Split(parts[1], "/")
		if len(hostDB) != 2 {
			return nil, fmt.Errorf("invalid connection string format")
		}

		hostPort := strings.Split(hostDB[0], ":")
		if len(hostPort) != 2 {
			return nil, fmt.Errorf("invalid connection string format")
		}

		host := hostPort[0]
		port := hostPort[1]
		dbname := hostDB[1]

		// TODO: This assumes database is on the host machine (local development)
		// In production, the database might be:
		// - Remote server (use actual hostname/IP from connection string)
		// - Same Docker network (use container name)
		// - Cloud service (use cloud endpoint)
		// Need to detect deployment context and handle accordingly
		if host == "localhost" || host == "127.0.0.1" {
			host = "host.docker.internal"
			log.Printf("Replaced localhost with host.docker.internal for Docker networking")
		}

		// Generate userlist.txt file for authentication
		configDir, err := generateUserlistFile(user, password)
		if err != nil {
			return nil, fmt.Errorf("failed to generate userlist.txt: %w", err)
		}

		defaultPoolSize := 20
		maxClientConn := 100
		reservePoolSize := 5

		// Pull image
		log.Printf("Pulling PgBouncer image...")
		if err := a.dockerClient.PullImage(ctx, "pgbouncer/pgbouncer:latest"); err != nil {
			return nil, fmt.Errorf("failed to pull PgBouncer image: %w", err)
		}

		// Create container
		log.Printf("Creating PgBouncer container: %s", a.containerName)

		portBindings := nat.PortMap{
			"6432/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "6432",
				},
			},
		}

		env := []string{
			fmt.Sprintf("DATABASES_HOST=%s", host),
			fmt.Sprintf("DATABASES_PORT=%s", port),
			fmt.Sprintf("DATABASES_USER=%s", user),
			fmt.Sprintf("DATABASES_DBNAME=%s", dbname),
			"PGBOUNCER_POOL_MODE=transaction",
			fmt.Sprintf("PGBOUNCER_DEFAULT_POOL_SIZE=%d", defaultPoolSize),
			fmt.Sprintf("PGBOUNCER_MAX_CLIENT_CONN=%d", maxClientConn),
			fmt.Sprintf("PGBOUNCER_RESERVE_POOL_SIZE=%d", reservePoolSize),
			"PGBOUNCER_AUTH_TYPE=md5",
			"PGBOUNCER_AUTH_FILE=/etc/pgbouncer/userlist.txt",
		}

		if password != "" {
			env = append(env, fmt.Sprintf("DATABASES_PASSWORD=%s", password))
		}

		containerConfig := &dockertypes.Config{
			Image: "pgbouncer/pgbouncer:latest",
			Env:   env,
			ExposedPorts: nat.PortSet{
				"6432/tcp": struct{}{},
			},
		}

		hostConfig := &dockertypes.HostConfig{
			PortBindings: portBindings,
			RestartPolicy: dockertypes.RestartPolicy{
				Name: "unless-stopped",
			},
			Binds: []string{
				fmt.Sprintf("%s:/etc/pgbouncer", configDir),
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
		log.Printf("Starting PgBouncer container...")
		if err := a.dockerClient.StartContainer(ctx, containerID); err != nil {
			a.dockerClient.RemoveContainer(ctx, containerID)
			return nil, fmt.Errorf("failed to start container: %w", err)
		}

		message = fmt.Sprintf("PgBouncer deployed as container '%s' on port 6432", a.containerName)
	}

	// Verify running
	log.Printf("Verifying PgBouncer is running...")
	time.Sleep(3 * time.Second)

	isRunning, err := a.dockerClient.IsContainerRunning(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check container status: %w", err)
	}

	if !isRunning {
		return nil, fmt.Errorf("container started but is not running - check logs with: docker logs %s", a.containerName)
	}

	log.Printf("PgBouncer is running on port 6432")

	endTime := time.Now()
	executionTimeMs := endTime.Sub(startTime).Milliseconds()

	result := &models.ActionResult{
		ActionID:        a.actionID,
		DetectionID:     a.detectionID,
		ActionType:      "deploy_pgbouncer",
		DatabaseID:      a.databaseID,
		Status:          models.StatusCompleted,
		Message:         message,
		CreatedAt:       startTime,
		Started:         &startTime,
		Completed:       &endTime,
		ExecutionTimeMs: executionTimeMs,
		Changes: map[string]interface{}{
			"container_name":  a.containerName,
			"container_id":    containerID,
			"pgbouncer_port":  6432,
			"pool_size":       20,
			"max_client_conn": 100,
			"instruction":     "Update your app's DB_CONNECTION_STRING to use port 6432 instead of 5432",
		},
		CanRollback: true,
		Rolledback:  false,
	}

	a.deployed = true
	return result, nil
}

func (a *DeployPgBouncerAction) Rollback(ctx context.Context) error {
	if !a.deployed {
		return fmt.Errorf("PgBouncer was not deployed, cannot rollback")
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

	log.Printf("PgBouncer container removed: %s", a.containerName)

	a.deployed = false
	return nil
}

func (a *DeployPgBouncerAction) Validate(ctx context.Context) error {
	// Check Docker is available
	if err := a.dockerClient.IsAvailable(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check database exists in Knowledge
	dbResp, err := a.knowledgeClient.GetDatabase(ctx, &pb.GetDatabaseRequest{
		DatabaseId: a.databaseID,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch database from Knowledge: %w", err)
	}

	if !dbResp.Found {
		return fmt.Errorf("database not found in Knowledge: %s", a.databaseID)
	}

	return nil
}

func (a *DeployPgBouncerAction) GetMetadata() *models.ActionMetadata {
	return &models.ActionMetadata{
		ActionID:     a.actionID,
		ActionType:   "deploy_pgbouncer",
		DatabaseID:   a.databaseID,
		DatabaseType: a.databaseType,
		CreatedAt:    time.Now(),
	}
}

// generateUserlistFile creates a userlist.txt file for PgBouncer authentication
func generateUserlistFile(user, password string) (string, error) {
	// Create temp directory for PgBouncer config
	configDir := "/tmp/pgbouncer-config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	userlistPath := filepath.Join(configDir, "userlist.txt")

	// Generate MD5 hash: "md5" + md5(password + username)
	hash := md5.Sum([]byte(password + user))
	md5Hash := fmt.Sprintf("md5%x", hash)

	// Write userlist.txt in format: "username" "md5hash"
	content := fmt.Sprintf("\"%s\" \"%s\"\n", user, md5Hash)

	if err := os.WriteFile(userlistPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write userlist.txt: %w", err)
	}

	log.Printf("Generated userlist.txt at: %s", userlistPath)
	return configDir, nil
}
