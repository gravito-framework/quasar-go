package probes

import (
	"strings"
	"sync" // Added for mutex

	"github.com/shirou/gopsutil/v3/process"
)

// LaravelWorkerDetail contains details for a single worker process
type LaravelWorkerDetail struct {
	PID      int32   `json:"pid"`
	Cmdline  string  `json:"cmdline"`
	Memory   uint64  `json:"memory"` // RSS in bytes
	CPU      float64 `json:"cpu"`    // Percent
	Status   string  `json:"status"` // "running", "sleeping", etc.
}

// LaravelWorkerStats contains information about running Laravel workers
type LaravelWorkerStats struct {
	WorkerCount int                   `json:"workerCount"`
	Roots       []string              `json:"roots"`
	Workers     []LaravelWorkerDetail `json:"workers"`
}


var (
	// workerProcessCache maintains state for process CPU calculations
	workerProcessCache = make(map[int32]*process.Process)
	cacheMutex         sync.Mutex
)

// GetLaravelWorkerStats scans the system for php artisan queue:work or artisan horizon processes
func GetLaravelWorkerStats() *LaravelWorkerStats {
	allProcs, err := process.Processes()
	if err != nil {
		return &LaravelWorkerStats{}
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	activePids := make(map[int32]bool)
	workers := []LaravelWorkerDetail{}
	rootsMap := make(map[string]bool)
	workerCount := 0

	for _, p := range allProcs {
		// Use cached process if available to get accurate CPU stats
		proc := p
		if cached, ok := workerProcessCache[p.Pid]; ok {
			proc = cached
		} else {
			// First time seeing this process, prime CPU calculation
			proc.CPUPercent()
			workerProcessCache[p.Pid] = proc
		}
		
		activePids[p.Pid] = true

		// We need to check cmdline.
		// Note: accessing Cmdline on a cached process is fine, it usually calls out to OS or uses cached value depending on impl.
		// gopsutil usually fetches live.
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}

		if strings.Contains(cmdline, "artisan") && (strings.Contains(cmdline, "queue:work") || strings.Contains(cmdline, "horizon")) {
			workerCount++

			var memRSS uint64 = 0
			memInfo, err := proc.MemoryInfo()
			if err == nil {
				memRSS = memInfo.RSS
			}

			// This should now return a non-zero value on subsequent calls
			cpuPercent, _ := proc.CPUPercent()
			
			status, _ := proc.Status()
			
			cwd, err := proc.Cwd()
			if err == nil && cwd != "" {
				rootsMap[cwd] = true
			}

			workers = append(workers, LaravelWorkerDetail{
				PID:     proc.Pid,
				Cmdline: cmdline,
				Memory:  memRSS,
				CPU:     cpuPercent,
				Status:  strings.Join(status, ","),
			})
		}
	}

	// Prune cache
	for pid := range workerProcessCache {
		if !activePids[pid] {
			delete(workerProcessCache, pid)
		}
	}

	roots := make([]string, 0, len(rootsMap))
	for root := range rootsMap {
		roots = append(roots, root)
	}

	return &LaravelWorkerStats{
		WorkerCount: workerCount,
		Roots:       roots,
		Workers:     workers,
	}
}
