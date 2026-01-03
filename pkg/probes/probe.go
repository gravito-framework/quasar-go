// Package probes provides interfaces and implementations for collecting metrics.
package probes

import "github.com/gravito-framework/quasar-go/pkg/types"

// SystemProbe collects system and process metrics (CPU, Memory, etc.)
type SystemProbe interface {
	GetMetrics() (*SystemMetrics, error)
}

// SystemMetrics contains the collected system information
type SystemMetrics struct {
	Language types.Language
	Version  string
	PID      int
	Hostname string
	Platform string
	Uptime   float64 // seconds
	CPU      types.CPUMetrics
	Memory   types.MemoryMetrics
}

// QueueProbe collects queue state snapshot
type QueueProbe interface {
	GetSnapshot() (*types.QueueSnapshot, error)
}
