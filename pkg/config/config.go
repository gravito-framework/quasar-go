// Package config handles configuration loading from environment variables and files.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the Quasar Agent
type Config struct {
	// Service identification
	Service string // Required: service name (e.g., "my-laravel-app")
	Name    string // Optional: custom node name (defaults to hostname)

	// Transport Redis (for sending heartbeats to Zenith)
	TransportRedisURL string

	// Monitor Redis (for inspecting local app queues)
	MonitorRedisURL string

	// Agent behavior
	Interval time.Duration // Heartbeat interval (default: 10s)

	// Queue monitoring configuration
	Queues []QueueConfig
}

// QueueConfig represents a queue to monitor
type QueueConfig struct {
	Name   string // Queue name
	Type   string // Type: "redis", "laravel", "bullmq"
	Prefix string // Optional key prefix
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		TransportRedisURL: "redis://localhost:6379",
		Interval:          10 * time.Second,
		Queues:            []QueueConfig{},
	}
}

// Load creates a Config from environment variables
func Load() *Config {
	cfg := DefaultConfig()

	// Required
	if v := os.Getenv("QUASAR_SERVICE"); v != "" {
		cfg.Service = v
	}

	// Optional name override
	if v := os.Getenv("QUASAR_NAME"); v != "" {
		cfg.Name = v
	}

	// Redis URLs
	if v := os.Getenv("QUASAR_TRANSPORT_REDIS_URL"); v != "" {
		cfg.TransportRedisURL = v
	} else if v := os.Getenv("QUASAR_REDIS_URL"); v != "" {
		// Legacy shorthand
		cfg.TransportRedisURL = v
	} else if v := os.Getenv("REDIS_URL"); v != "" {
		// Common convention
		cfg.TransportRedisURL = v
	}

	if v := os.Getenv("QUASAR_MONITOR_REDIS_URL"); v != "" {
		cfg.MonitorRedisURL = v
	}

	// Interval
	if v := os.Getenv("QUASAR_INTERVAL"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.Interval = time.Duration(seconds) * time.Second
		}
	}

	// Queue monitoring (comma-separated: name:type,name:type)
	// Example: QUASAR_QUEUES=default:laravel,emails:redis
	if v := os.Getenv("QUASAR_QUEUES"); v != "" {
		queues := parseQueues(v)
		cfg.Queues = append(cfg.Queues, queues...)
	}

	return cfg
}

// parseQueues parses queue configuration string
// Format: "name:type,name:type" or "name" (defaults to laravel)
func parseQueues(s string) []QueueConfig {
	var queues []QueueConfig

	// Split by comma
	parts := splitAndTrim(s, ",")
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Split by colon
		segments := splitAndTrim(part, ":")
		if len(segments) == 0 {
			continue
		}

		qc := QueueConfig{
			Name: segments[0],
			Type: "laravel", // default
		}

		if len(segments) >= 2 {
			qc.Type = segments[1]
		}

		if len(segments) >= 3 {
			qc.Prefix = segments[2]
		}

		queues = append(queues, qc)
	}

	return queues
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	start := 0

	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])

	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Service == "" {
		return &ConfigError{Field: "Service", Message: "service name is required (set QUASAR_SERVICE)"}
	}
	if c.TransportRedisURL == "" {
		return &ConfigError{Field: "TransportRedisURL", Message: "transport Redis URL is required"}
	}
	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + ": " + e.Message
}
