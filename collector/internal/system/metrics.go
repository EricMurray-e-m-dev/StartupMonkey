package system

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type Metrics struct {
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	MemoryUsedBytes    uint64
	MemoryTotalBytes   uint64
	LoadAvg1m          float64
	LoadAvg5m          float64
	LoadAvg15m         float64
}

func Collect() (*Metrics, error) {
	m := &Metrics{}

	// CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		m.CPUUsagePercent = cpuPercent[0]
	}

	// Memory
	memStats, err := mem.VirtualMemory()
	if err == nil {
		m.MemoryUsagePercent = memStats.UsedPercent
		m.MemoryUsedBytes = memStats.Used
		m.MemoryTotalBytes = memStats.Total
	}

	// Load average
	loadStats, err := load.Avg()
	if err == nil {
		m.LoadAvg1m = loadStats.Load1
		m.LoadAvg5m = loadStats.Load5
		m.LoadAvg15m = loadStats.Load15
	}

	return m, nil
}

// ToExtendedMetrics converts to map for inclusion in RawMetrics
func (m *Metrics) ToExtendedMetrics() map[string]float64 {
	return map[string]float64{
		"system.cpu_usage_percent":    m.CPUUsagePercent,
		"system.memory_usage_percent": m.MemoryUsagePercent,
		"system.memory_used_bytes":    float64(m.MemoryUsedBytes),
		"system.memory_total_bytes":   float64(m.MemoryTotalBytes),
		"system.load_1m":              m.LoadAvg1m,
		"system.load_5m":              m.LoadAvg5m,
		"system.load_15m":             m.LoadAvg15m,
	}
}
