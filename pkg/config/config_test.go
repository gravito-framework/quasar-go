package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original env
	originalEnv := make(map[string]string)
	envVars := []string{
		"QUASAR_SERVICE",
		"QUASAR_NAME",
		"QUASAR_REDIS_URL",
		"QUASAR_TRANSPORT_REDIS_URL",
		"QUASAR_MONITOR_REDIS_URL",
		"QUASAR_INTERVAL",
		"QUASAR_QUEUES",
	}
	
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	
	// Restore env after test
	defer func() {
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("defaults", func(t *testing.T) {
		cfg := Load()
		
		if cfg.TransportRedisURL != "redis://localhost:6379" {
			t.Errorf("Expected default TransportRedisURL, got %s", cfg.TransportRedisURL)
		}
		
		if cfg.Interval != 10*time.Second {
			t.Errorf("Expected default interval 10s, got %v", cfg.Interval)
		}
	})

	t.Run("from environment", func(t *testing.T) {
		os.Setenv("QUASAR_SERVICE", "test-service")
		os.Setenv("QUASAR_NAME", "test-node")
		os.Setenv("QUASAR_TRANSPORT_REDIS_URL", "redis://zenith:6379")
		os.Setenv("QUASAR_MONITOR_REDIS_URL", "redis://app:6379")
		os.Setenv("QUASAR_INTERVAL", "5")
		os.Setenv("QUASAR_QUEUES", "default:laravel,emails:redis")
		
		cfg := Load()
		
		if cfg.Service != "test-service" {
			t.Errorf("Expected service test-service, got %s", cfg.Service)
		}
		
		if cfg.Name != "test-node" {
			t.Errorf("Expected name test-node, got %s", cfg.Name)
		}
		
		if cfg.TransportRedisURL != "redis://zenith:6379" {
			t.Errorf("Expected TransportRedisURL redis://zenith:6379, got %s", cfg.TransportRedisURL)
		}
		
		if cfg.MonitorRedisURL != "redis://app:6379" {
			t.Errorf("Expected MonitorRedisURL redis://app:6379, got %s", cfg.MonitorRedisURL)
		}
		
		if cfg.Interval != 5*time.Second {
			t.Errorf("Expected interval 5s, got %v", cfg.Interval)
		}
		
		if len(cfg.Queues) != 2 {
			t.Fatalf("Expected 2 queues, got %d", len(cfg.Queues))
		}
		
		if cfg.Queues[0].Name != "default" || cfg.Queues[0].Type != "laravel" {
			t.Errorf("Expected queue[0] default:laravel, got %s:%s", cfg.Queues[0].Name, cfg.Queues[0].Type)
		}
		
		if cfg.Queues[1].Name != "emails" || cfg.Queues[1].Type != "redis" {
			t.Errorf("Expected queue[1] emails:redis, got %s:%s", cfg.Queues[1].Name, cfg.Queues[1].Type)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("missing service", func(t *testing.T) {
		cfg := &Config{
			TransportRedisURL: "redis://localhost:6379",
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for missing service")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			Service:           "test-service",
			TransportRedisURL: "redis://localhost:6379",
		}
		
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no validation error, got %v", err)
		}
	})
}

func TestParseQueues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []QueueConfig
	}{
		{
			name:  "single queue with type",
			input: "default:laravel",
			expected: []QueueConfig{
				{Name: "default", Type: "laravel"},
			},
		},
		{
			name:  "multiple queues",
			input: "default:laravel,emails:redis,jobs:bullmq",
			expected: []QueueConfig{
				{Name: "default", Type: "laravel"},
				{Name: "emails", Type: "redis"},
				{Name: "jobs", Type: "bullmq"},
			},
		},
		{
			name:  "queue without type (defaults to laravel)",
			input: "default",
			expected: []QueueConfig{
				{Name: "default", Type: "laravel"},
			},
		},
		{
			name:  "with prefix",
			input: "default:laravel:custom_prefix",
			expected: []QueueConfig{
				{Name: "default", Type: "laravel", Prefix: "custom_prefix"},
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []QueueConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseQueues(tt.input)
			
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d queues, got %d", len(tt.expected), len(result))
			}
			
			for i, expected := range tt.expected {
				if result[i].Name != expected.Name {
					t.Errorf("Queue[%d] name: expected %s, got %s", i, expected.Name, result[i].Name)
				}
				if result[i].Type != expected.Type {
					t.Errorf("Queue[%d] type: expected %s, got %s", i, expected.Type, result[i].Type)
				}
				if result[i].Prefix != expected.Prefix {
					t.Errorf("Queue[%d] prefix: expected %s, got %s", i, expected.Prefix, result[i].Prefix)
				}
			}
		})
	}
}
