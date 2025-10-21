package engine

import (
	"log"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/detector"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
)

// Engine populated of detectors
type Engine struct {
	detectors []detector.Detector
}

// Create a new detection engine
func NewEngine() *Engine {
	return &Engine{
		detectors: make([]detector.Detector, 0),
	}
}

// Add new detector to the engine
func (e *Engine) RegisterDetector(d detector.Detector) {
	e.detectors = append(e.detectors, d)
	log.Printf("Registered detector: %s (category: %s)", d.Name(), d.Category())
}

// Runs all detectors on provided metrics snapshot from collector
func (e *Engine) RunDetectors(snapshot *normaliser.NormalisedMetrics) []*models.Detection {
	var detections []*models.Detection

	for _, det := range e.detectors {
		if detection := det.Detect(snapshot); detection != nil {
			log.Printf("Detection [%s] %s - %s", detection.Severity, det.Name(), detection.Title)
			detections = append(detections, detection)
		}
	}

	if len(detections) == 0 {
		log.Printf("No issues detected (database: %s)", snapshot.DatabaseID)
	} else {
		log.Printf("Found %d issues in database: %s", len(detections), snapshot.DatabaseID)
	}

	return detections
}

// Returns list of registered detectors
func (e *Engine) GetRegisteredDetectors() []string {
	names := make([]string, len(e.detectors))
	for i, det := range e.detectors {
		names[i] = det.Name()
	}
	return names
}
