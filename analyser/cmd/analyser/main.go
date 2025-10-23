package main

import (
	"log"
	"net"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/grpc"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Printf("Starting Analyser service...")

	detectionEngine := engine.NewEngine()

	detectionEngine.RegisterDetector(detector.NewCacheMissDetector())
	detectionEngine.RegisterDetector(detector.NewConnectionPoolDetection())
	detectionEngine.RegisterDetector(detector.NewHighLatencyDetector())
	detectionEngine.RegisterDetector(detector.NewMissingIndexDetector())

	log.Printf("Detection engine initialised with %d detectors", len(detectionEngine.GetRegisteredDetectors()))

	metricServer := grpcserver.NewMetricsServer(detectionEngine)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMetricsServiceServer(grpcServer, metricServer)

	reflection.Register(grpcServer)

	log.Printf("Analyser listening on 50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
