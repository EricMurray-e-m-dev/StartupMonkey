package main

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/eventbus"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/grpc"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/health"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/knowledge"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Printf("Starting Analyser service...")

	envPath := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(envPath)

	detectionEngine := engine.NewEngine()

	detectionEngine.RegisterDetector(detector.NewCacheMissDetector())
	detectionEngine.RegisterDetector(detector.NewConnectionPoolDetection())
	detectionEngine.RegisterDetector(detector.NewHighLatencyDetector())
	detectionEngine.RegisterDetector(detector.NewMissingIndexDetector())

	log.Printf("Detection engine initialised with %d detectors", len(detectionEngine.GetRegisteredDetectors()))

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := eventbus.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer publisher.Close()
	log.Printf("Connected to NATS at %s", natsURL)

	knowledgeAddr := os.Getenv("KNOWLEDGE_ADDRESS")
	if knowledgeAddr == "" {
		knowledgeAddr = "localhost:50053"
	}

	knowledgeClient, err := knowledge.NewKnowledgeClient(knowledgeAddr)
	if err != nil {
		log.Fatalf("Failed to connect to brain: %v", err)
	}
	defer knowledgeClient.Close()

	subscriber, err := eventbus.NewSubscriber(natsURL, knowledgeClient)
	if err != nil {
		log.Fatalf("failed to create NATS subscriber: %v", err)
	}
	defer subscriber.Close()

	metricServer := grpcserver.NewMetricsServer(detectionEngine, publisher, knowledgeClient)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMetricsServiceServer(grpcServer, metricServer)

	reflection.Register(grpcServer)

	log.Printf("Analyser listening on 50051")
	health.StartHealthCheckServer("8081")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
