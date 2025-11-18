package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
	"github.com/joho/godotenv"
)

func main() {
	log.Printf("Starting up brain")

	envPath := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(envPath)

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisClient, err := redis.NewClient(redisAddr, redisPassword, 0)
	if err != nil {
		log.Fatalf("Failed to create redis client: %v", err)
	}
	defer redisClient.Close()

	healthServer := health.NewHealthServer(redisClient)
	go func() {
		log.Printf("Health server listening on: 8083")
		if err := healthServer.Start(":8083"); err != nil {
			log.Fatalf("Health check on brain failed: %v", err)
		}
	}()

	log.Printf("Brain ready")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutting down brain")
	healthServer.Shutdown(context.Background())

}
