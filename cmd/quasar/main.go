// Quasar Agent - The Gravito Infrastructure Monitor
//
// A standalone daemon/sidecar for monitoring system resources and application queues.
// Designed for PHP/Laravel, Legacy, and Polyglot environments.
//
// Usage:
//
//	QUASAR_SERVICE=my-app QUASAR_REDIS_URL=redis://localhost:6379 quasar
//
// Or with a config file:
//
//	quasar --config /etc/quasar/config.yaml
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gravito-framework/quasar-go/pkg/agent"
	"github.com/gravito-framework/quasar-go/pkg/config"
	"github.com/gravito-framework/quasar-go/pkg/probes/queue"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Print banner
	fmt.Printf(`
   ██████  ██    ██  █████  ███████  █████  ██████  
  ██    ██ ██    ██ ██   ██ ██      ██   ██ ██   ██ 
  ██    ██ ██    ██ ███████ ███████ ███████ ██████  
  ██ ▄▄ ██ ██    ██ ██   ██      ██ ██   ██ ██   ██ 
   ██████   ██████  ██   ██ ███████ ██   ██ ██   ██ 
      ▀▀                                            
  🌌 Quasar Agent %s (%s)
  The brightest signal in your infrastructure.

`, version, commit[:min(7, len(commit))])

	// Load configuration
	cfg := config.Load()

	// Handle --help or --version
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		case "--version", "-v":
			fmt.Printf("quasar %s (commit: %s, built: %s)\n", version, commit, date)
			os.Exit(0)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration error", "error", err)
		fmt.Println("\nRun 'quasar --help' for usage information.")
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create agent
	a, err := agent.New(cfg, agent.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create agent", "error", err)
		os.Exit(1)
	}

	// Add queue probes from config
	monitorClient := a.GetMonitorClient()
	for _, q := range cfg.Queues {
		switch q.Type {
		case "laravel":
			if q.Prefix != "" {
				a.AddQueueProbe(queue.NewLaravelProbeWithPrefix(monitorClient, q.Name, q.Prefix))
			} else {
				a.AddQueueProbe(queue.NewLaravelProbe(monitorClient, q.Name))
			}
			logger.Info("Monitoring Laravel queue", "name", q.Name)
		case "redis":
			a.AddQueueProbe(queue.NewRedisListProbe(monitorClient, q.Name))
			logger.Info("Monitoring Redis queue", "name", q.Name)
		}
	}

	// Start agent
	if err := a.Start(ctx); err != nil {
		logger.Error("Failed to start agent", "error", err)
		os.Exit(1)
	}

	// Enable remote control
	if err := a.EnableRemoteControl(ctx); err != nil {
		logger.Warn("Failed to enable remote control", "error", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("Received shutdown signal", "signal", sig)

	// Graceful shutdown
	cancel()
	if err := a.Stop(context.Background()); err != nil {
		logger.Error("Shutdown error", "error", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Usage: quasar [options]

Quasar is the Gravito infrastructure monitoring agent. It collects system
metrics (CPU, RAM) and queue status, sending them to Zenith for visualization.

Environment Variables:
  QUASAR_SERVICE              (Required) Service name identifier
  QUASAR_NAME                 Custom node name (default: hostname)
  QUASAR_REDIS_URL            Redis URL for Zenith transport (default: redis://localhost:6379)
  QUASAR_TRANSPORT_REDIS_URL  Same as QUASAR_REDIS_URL
  QUASAR_MONITOR_REDIS_URL    Redis URL for local app queue monitoring
  QUASAR_INTERVAL             Heartbeat interval in seconds (default: 10)

Options:
  -h, --help      Show this help message
  -v, --version   Show version information

Examples:
  # Basic usage (monitor system only)
  QUASAR_SERVICE=my-laravel-app quasar

  # With separate transport and monitor Redis
  QUASAR_SERVICE=my-app \
  QUASAR_TRANSPORT_REDIS_URL=redis://zenith-redis:6379 \
  QUASAR_MONITOR_REDIS_URL=redis://localhost:6379 \
  quasar

  # Docker usage
  docker run -e QUASAR_SERVICE=my-app gravito/quasar-agent

For more information, visit: https://github.com/gravito-framework/quasar
`)
}
