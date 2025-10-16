package orchestrator

import (
	"log"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/config"
	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
)

type Orchestrator struct {
	config  *config.Config
	adapter adapter.MetricAdapter
	client  *grpcclient.MetricsClient
}

func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config: cfg,
	}
}

func (o *Orchestrator) Start() error {
	log.Printf("Starting Collector Orchestrator...")

	var err error
	o.adapter, err = adapter.NewAdapter(o.config.DBAdapter, o.config.DBConnectionString)
	if err != nil {
		return err
	}

	log.Printf("Connecting to database (%s)", o.config.DBAdapter)
	if err := o.adapter.Connect(); err != nil {
		return err
	}
	log.Printf("Database Connected.")

	if err := o.adapter.HealthCheck(); err != nil {
		return err
	}
	log.Printf("Database healthy.")

	o.client = grpcclient.NewMetricsClient(o.config.AnalyserAddress)
	if err := o.client.Connect(); err != nil {
		return err
	}
	log.Printf("Connected to Analyser at %s", o.config.AnalyserAddress)

	return nil
}
