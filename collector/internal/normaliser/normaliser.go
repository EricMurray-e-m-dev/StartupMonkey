package normaliser

import "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/adapter"

type Normaliser interface {
	Normalise(raw *adapter.RawMetrics) (*NormalisedMetrics, error)
}

func NewNormaliser(databaseType string) Normaliser {
	switch databaseType {
	case "postgres", "postgresql":
		return &PostgresNormaliser{}
	}
}
