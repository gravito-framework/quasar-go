// Package agent provides the QuasarAgent core implementation.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gravito-framework/quasar-go/pkg/config"
	"github.com/gravito-framework/quasar-go/pkg/probes"
	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "gravito:quasar:node:"
	keyTTL    = 30 * time.Second
)

// Agent is the main Quasar monitoring agent
type Agent struct {
	config *config.Config
	logger *slog.Logger

	// Redis connections
	transportRedis *redis.Client // For sending heartbeats to Zenith
	monitorRedis   *redis.Client // For inspecting local app queues (optional)

	// Probes
	systemProbe probes.SystemProbe
	queueProbes []probes.QueueProbe

	// Command listener (for remote control)
	commandListener *CommandListener

	// State
	nodeID   string
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// Option is a functional option for configuring the Agent
type Option func(*Agent)

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) Option {
	return func(a *Agent) {
		a.logger = logger
	}
}

// WithSystemProbe sets a custom system probe
func WithSystemProbe(probe probes.SystemProbe) Option {
	return func(a *Agent) {
		a.systemProbe = probe
	}
}

// New creates a new Quasar Agent
func New(cfg *config.Config, opts ...Option) (*Agent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	a := &Agent{
		config:      cfg,
		logger:      slog.Default(),
		queueProbes: []probes.QueueProbe{},
		stopChan:    make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	// Parse transport Redis URL
	transportOpts, err := redis.ParseURL(cfg.TransportRedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid transport redis URL: %w", err)
	}
	a.transportRedis = redis.NewClient(transportOpts)

	// Parse monitor Redis URL if provided
	if cfg.MonitorRedisURL != "" {
		monitorOpts, err := redis.ParseURL(cfg.MonitorRedisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid monitor redis URL: %w", err)
		}
		a.monitorRedis = redis.NewClient(monitorOpts)
	}

	// Create default system probe if not provided
	if a.systemProbe == nil {
		probe, err := probes.NewGoSystemProbe()
		if err != nil {
			return nil, fmt.Errorf("failed to create system probe: %w", err)
		}
		a.systemProbe = probe
	}

	return a, nil
}

// Start begins the agent's heartbeat loop
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent already running")
	}
	a.running = true
	a.mu.Unlock()

	// Test transport connection (non-fatal)
	if err := a.transportRedis.Ping(ctx).Err(); err != nil {
		a.logger.Warn("⚠️ Failed to connect to transport Redis, will retry in background", "error", err)
	}

	// Test monitor connection if provided (non-fatal)
	if a.monitorRedis != nil {
		if err := a.monitorRedis.Ping(ctx).Err(); err != nil {
			a.logger.Warn("⚠️ Failed to connect to monitor Redis, stats might be missing", "error", err)
		}
	}

	a.logger.Info("Quasar Agent started",
		"service", a.config.Service,
		"interval", a.config.Interval,
	)

	// Initial tick to set nodeID
	if err := a.tick(ctx); err != nil {
		a.logger.Error("Initial heartbeat failed", "error", err)
	}

	// Start heartbeat loop
	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.mu.Unlock()

	close(a.stopChan)

	// Stop command listener if active
	if a.commandListener != nil {
		if err := a.commandListener.Stop(ctx); err != nil {
			a.logger.Error("Failed to stop command listener", "error", err)
		}
	}

	// Wait for goroutines
	a.wg.Wait()

	// Stop system probe if it has a Stop method
	if probe, ok := a.systemProbe.(*probes.GoSystemProbe); ok {
		probe.Stop()
	}

	// Close Redis connections
	if err := a.transportRedis.Close(); err != nil {
		a.logger.Error("Failed to close transport Redis", "error", err)
	}
	if a.monitorRedis != nil {
		if err := a.monitorRedis.Close(); err != nil {
			a.logger.Error("Failed to close monitor Redis", "error", err)
		}
	}

	a.logger.Info("Quasar Agent stopped")
	return nil
}

// NodeID returns the current node identifier
func (a *Agent) NodeID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.nodeID
}

// AddQueueProbe adds a queue probe for monitoring
func (a *Agent) AddQueueProbe(probe probes.QueueProbe) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queueProbes = append(a.queueProbes, probe)
}

// EnableRemoteControl enables the command listener for Zenith commands
func (a *Agent) EnableRemoteControl(ctx context.Context) error {
	if a.monitorRedis == nil {
		return fmt.Errorf("monitor Redis connection required for remote control")
	}

	a.mu.RLock()
	nodeID := a.nodeID
	a.mu.RUnlock()

	if nodeID == "" {
		return fmt.Errorf("agent not started (nodeID unknown)")
	}

	// Create a dedicated subscriber connection
	subscriberOpts, _ := redis.ParseURL(a.config.TransportRedisURL)
	subscriberRedis := redis.NewClient(subscriberOpts)

	a.commandListener = NewCommandListener(
		subscriberRedis,
		a.config.Service,
		nodeID,
		a.logger,
	)

	if err := a.commandListener.Start(ctx, a.monitorRedis); err != nil {
		return fmt.Errorf("failed to start command listener: %w", err)
	}

	a.logger.Info("🎮 Remote control enabled", "nodeId", nodeID)
	return nil
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.tick(ctx); err != nil {
				a.logger.Error("Heartbeat failed", "error", err)
			}
		}
	}
}

func (a *Agent) tick(ctx context.Context) error {
	// Collect system metrics
	metrics, err := a.systemProbe.GetMetrics()
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Generate node ID
	hostname := a.config.Name
	if hostname == "" {
		hostname = metrics.Hostname
	}
	nodeID := fmt.Sprintf("%s-%d", hostname, metrics.PID)

	// Update cached nodeID
	a.mu.Lock()
	a.nodeID = nodeID
	a.mu.Unlock()

	// Collect queue snapshots
	var queues []types.QueueSnapshot
	a.mu.RLock()
	queueProbes := a.queueProbes
	a.mu.RUnlock()

	for _, probe := range queueProbes {
		snapshot, err := probe.GetSnapshot()
		if err != nil {
			a.logger.Warn("Queue probe failed", "error", err)
			continue
		}
		queues = append(queues, *snapshot)
	}

	// Check connection health
	var agentErrors []string
	agentStatus := "online"

	if err := a.transportRedis.Ping(ctx).Err(); err != nil {
		// We can't actually SEND this if transport is down, 
		// but we track it for local logging and future recovery
		agentStatus = "error"
		agentErrors = append(agentErrors, "transport_redis_offline")
	}

	if a.monitorRedis != nil {
		if err := a.monitorRedis.Ping(ctx).Err(); err != nil {
			agentStatus = "degraded"
			agentErrors = append(agentErrors, "monitor_redis_offline")
		}
	}

	// Build payload
	payload := types.HeartbeatPayload{
		ID:       nodeID,
		Service:  a.config.Service,
		Language: metrics.Language,
		Version:  metrics.Version,
		PID:      metrics.PID,
		Hostname: hostname,
		Platform: metrics.Platform,
		CPU:      metrics.CPU,
		Memory:   metrics.Memory,
		Queues:   queues,
		Runtime: types.RuntimeInfo{
			Uptime:    metrics.Uptime,
			Framework: "Quasar",
			Status:    agentStatus,
			Errors:    agentErrors,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	// Serialize and send to Redis
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	key := keyPrefix + a.config.Service + ":" + nodeID
	if err := a.transportRedis.Set(ctx, key, data, keyTTL).Err(); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	a.logger.Debug("Heartbeat sent", "key", key, "cpu", metrics.CPU.Process)
	return nil
}
