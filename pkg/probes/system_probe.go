package probes

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// GoSystemProbe implements SystemProbe using gopsutil
type GoSystemProbe struct {
	startTime time.Time
	proc      *process.Process

	// CPU sampling
	mu               sync.RWMutex
	lastCPUTimes     cpu.TimesStat
	lastSampleTime   time.Time
	cachedCPUPercent float64
	stopSampler      chan struct{}
	isDarwin         bool
}

// NewGoSystemProbe creates a new system probe for Go processes
func NewGoSystemProbe() (*GoSystemProbe, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}

	probe := &GoSystemProbe{
		startTime:   time.Now(),
		proc:        p,
		stopSampler: make(chan struct{}),
		isDarwin:    runtime.GOOS == "darwin",
	}

	// Initialize CPU baseline
	times, err := cpu.Times(false)
	if err == nil && len(times) > 0 {
		probe.lastCPUTimes = times[0]
		probe.lastSampleTime = time.Now()
	}

	// Wait 1 second and take first sample to initialize cachedCPUPercent
	time.Sleep(1 * time.Second)
	probe.sampleCPU()

	// Start background CPU sampler (samples every 1 second)
	go probe.cpuSampler()

	return probe, nil
}

// cpuSampler runs in background to sample CPU usage every second
func (p *GoSystemProbe) cpuSampler() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.sampleCPU()
		case <-p.stopSampler:
			return
		}
	}
}

// sampleCPU takes a CPU sample and calculates usage
func (p *GoSystemProbe) sampleCPU() {
	times, err := cpu.Times(false)

	p.mu.Lock()
	defer p.mu.Unlock()

	if err == nil && len(times) > 0 {
		current := times[0]
		now := time.Now()

		deltaTotal := current.Total() - p.lastCPUTimes.Total()
		deltaIdle := current.Idle - p.lastCPUTimes.Idle

		if deltaTotal > 0 {
			p.cachedCPUPercent = round(100*(deltaTotal-deltaIdle)/deltaTotal, 2)
		}

		p.lastCPUTimes = current
		p.lastSampleTime = now
	} else if p.isDarwin {
		// Fallback for Darwin when CGO is disabled or cpu.Times fails
		if val, err := p.getDarwinSystemCPU(); err == nil {
			p.cachedCPUPercent = val
		}
	}
}

// getDarwinSystemCPU parses 'top' output on macOS
func (p *GoSystemProbe) getDarwinSystemCPU() (float64, error) {
	// Run top in logging mode, 1 sample, 0 processes
	// Output format: "CPU usage: 12.34% user, 5.67% sys, 81.99% idle"
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "CPU usage:") {
			// Basic parsing without regex for performance
			// Expected: CPU usage: 14.86% user, 8.41% sys, 76.71% idle
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				continue
			}

			// Parse user
			userPart := strings.TrimSpace(strings.TrimPrefix(parts[0], "CPU usage:"))
			userVal := 0.0
			fmt.Sscanf(userPart, "%f%%", &userVal)

			// Parse sys
			sysPart := strings.TrimSpace(parts[1])
			sysVal := 0.0
			fmt.Sscanf(sysPart, "%f%%", &sysVal)

			return round(userVal+sysVal, 2), nil
		}
	}

	return 0, fmt.Errorf("could not parse top output")
}

// Stop stops the CPU sampler
func (p *GoSystemProbe) Stop() {
	close(p.stopSampler)
}

// GetMetrics collects current system and process metrics
func (p *GoSystemProbe) GetMetrics() (*SystemMetrics, error) {
	hostname, _ := os.Hostname()

	// CPU metrics
	cpuMetrics, err := p.getCPUMetrics()
	if err != nil {
		return nil, err
	}

	// Memory metrics
	memMetrics, err := p.getMemoryMetrics()
	if err != nil {
		return nil, err
	}

	return &SystemMetrics{
		Language: types.LangGo,
		Version:  runtime.Version(),
		PID:      os.Getpid(),
		Hostname: hostname,
		Platform: runtime.GOOS,
		Uptime:   time.Since(p.startTime).Seconds(),
		CPU:      *cpuMetrics,
		Memory:   *memMetrics,
	}, nil
}

func (p *GoSystemProbe) getCPUMetrics() (*types.CPUMetrics, error) {
	// System CPU: Use cached value from background sampler
	p.mu.RLock()
	systemPercent := p.cachedCPUPercent
	p.mu.RUnlock()

	// Core count - fallback to runtime.NumCPU()
	cores := runtime.NumCPU()
	if c, err := cpu.Counts(true); err == nil && c > 0 {
		cores = c
	}

	// Process CPU - use fallback if not available
	procPercent := 0.0
	if pct, err := p.proc.CPUPercent(); err == nil {
		// gopsutil returns percentage of ONE core (up to 100 * cores)
		// We normalize it to total system capacity (0-100)
		procPercent = round(pct/float64(cores), 2)
	}

	return &types.CPUMetrics{
		System:  systemPercent,
		Process: procPercent,
		Cores:   cores,
	}, nil
}

// round rounds a float64 to n decimal places
func round(val float64, decimals int) float64 {
	shift := float64(1)
	for i := 0; i < decimals; i++ {
		shift *= 10
	}
	return float64(int(val*shift+0.5)) / shift
}

func (p *GoSystemProbe) getMemoryMetrics() (*types.MemoryMetrics, error) {
	// System memory - use fallback if not available
	systemMem := types.SystemMemory{
		Total: 0,
		Free:  0,
		Used:  0,
	}

	if v, err := mem.VirtualMemory(); err == nil {
		systemMem.Total = v.Total
		systemMem.Free = v.Available
		systemMem.Used = v.Used
	}

	// Process memory - use fallback if not available
	processMem := types.ProcessMemory{
		RSS:       0,
		HeapTotal: 0,
		HeapUsed:  0,
	}

	if memInfo, err := p.proc.MemoryInfo(); err == nil {
		processMem.RSS = memInfo.RSS
		processMem.HeapTotal = memInfo.RSS
		processMem.HeapUsed = memInfo.RSS
	} else {
		// Fallback to Go runtime memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		processMem.RSS = m.Sys
		processMem.HeapTotal = m.HeapSys
		processMem.HeapUsed = m.HeapAlloc
	}

	return &types.MemoryMetrics{
		System:  systemMem,
		Process: processMem,
	}, nil
}
