package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/eventbus"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/health"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

func main() {
	log.Printf("Starting Executor Service")

	detectionHandler := handler.NewDetectionHandler()
	log.Printf("Detection Handler intialised")

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "http://localhost:4222"
	}

	subscriber, err := eventbus.NewSubscriber(natsURL, detectionHandler)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer subscriber.Close()

	if err := subscriber.Start(); err != nil {
		log.Fatalf("Failed to start subscriber: %v", err)
	}

	grpcPort := "50052"

	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	executorServer := grpcserver.NewExecutorSever()
	pb.RegisterExecutorServiceServer(grpcServer, executorServer)

	log.Printf("Executor gRPC server listening on :%s", grpcPort)
	health.StartHealthCheckServer("8082")

	// Handle shutdown on signal
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Printf("Shutting down executor service")
		grpcServer.GracefulStop()
		subscriber.Close()
		log.Printf("Executor service stopped successfully")
		os.Exit(0)
	}()

	log.Printf("Executor service ready")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to server gRPC: %v", err)
	}
}
