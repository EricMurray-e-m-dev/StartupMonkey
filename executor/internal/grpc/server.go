package grpc

import (
	"context"
	"log"
	"time"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

type ExecutorServer struct {
	pb.UnimplementedExecutorServiceServer
}

func NewExecutorSever() *ExecutorServer {
	return &ExecutorServer{}
}

func (s *ExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	uptime := time.Since(startTime).Seconds()

	log.Printf("Health check requested (uptime: %.0fs)", uptime)

	return &pb.HealthResponse{
		Status:        "healthy",
		UptimeSeconds: int64(uptime),
	}, nil
}
func (s *ExecutorServer) GetActionStatus(ctx context.Context, req *pb.ActionStatusRequest) (*pb.ActionStatusResponse, error) {
	// TODO: Wire to handler
	return &pb.ActionStatusResponse{
		ActionId: req.ActionId,
		Status:   "TODO",
		Message:  "TODO",
	}, nil
}

func (s *ExecutorServer) ListPendingActions(ctx context.Context, req *pb.ListRequest) (*pb.ActionList, error) {
	// TODO: Wire to handler
	return &pb.ActionList{
		Actions:    []*pb.ActionStatusResponse{},
		TotalCount: 0,
	}, nil
}
