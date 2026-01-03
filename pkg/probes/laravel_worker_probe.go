package probes

import (
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// LaravelWorkerStats contains information about running Laravel workers
type LaravelWorkerStats struct {
	WorkerCount int      `json:"workerCount"`
	Roots       []string `json:"roots"`
}

// GetLaravelWorkerStats scans the system for php artisan queue:work or artisan horizon processes
func GetLaravelWorkerStats() *LaravelWorkerStats {
	procs, err := process.Processes()
	if err != nil {
		return &LaravelWorkerStats{}
	}

	rootsMap := make(map[string]bool)
	workerCount := 0

	for _, p := range procs {
		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}

		// Check for Laravel queue worker or horizon
		if strings.Contains(cmdline, "artisan") && (strings.Contains(cmdline, "queue:work") || strings.Contains(cmdline, "horizon")) {
			workerCount++
			cwd, err := p.Cwd()
			if err == nil && cwd != "" {
				rootsMap[cwd] = true
			}
		}
	}

	roots := make([]string, 0, len(rootsMap))
	for root := range rootsMap {
		roots = append(roots, root)
	}

	return &LaravelWorkerStats{
		WorkerCount: workerCount,
		Roots:       roots,
	}
}
