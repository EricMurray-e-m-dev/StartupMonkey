package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"github.com/joho/godotenv"
	grpclib "google.golang.org/grpc"
)

func main() {
	log.Printf("Starting Brain...")

	// Load environment variables from ROOT .env
	envPath := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(envPath)

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Create Redis client
	redisClient, err := redis.NewClient(redisAddr, redisPassword, 0)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Start gRPC server
	grpcServer := grpclib.NewServer()
	pb.RegisterKnowledgeServiceServer(grpcServer, grpc.NewKnowledgeServer(redisClient))

	listener, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}

	go func() {
		log.Printf("Knowledge gRPC server listening on :50053")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start health check server
	healthServer := health.NewHealthServer(redisClient)
	go func() {
		log.Printf("Health check listening on :8083")
		if err := healthServer.Start(":8083"); err != nil {
			log.Fatalf("Health check server failed: %v", err)
		}
	}()

	log.Printf("Brain ready")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutting down Brain...")
	grpcServer.GracefulStop()
	healthServer.Shutdown(context.Background())
}
