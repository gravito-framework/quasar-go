// Package types defines shared types for the Quasar Agent.
// These types mirror the TypeScript SDK for protocol compatibility.
package types

import "time"

// Language represents the runtime/language type
type Language string

const (
	LangNode   Language = "node"
	LangBun    Language = "bun"
	LangDeno   Language = "deno"
	LangPHP    Language = "php"
	LangGo     Language = "go"
	LangPython Language = "python"
	LangOther  Language = "other"
)

// QueueDriver represents the queue driver type
type QueueDriver string

const (
	DriverRedis    QueueDriver = "redis"
	DriverSQS      QueueDriver = "sqs"
	DriverRabbitMQ QueueDriver = "rabbitmq"
)

// QueueSize contains queue depth metrics
type QueueSize struct {
	Waiting int64 `json:"waiting"`
	Active  int64 `json:"active"`
	Failed  int64 `json:"failed"`
	Delayed int64 `json:"delayed"`
}

// QueueThroughput contains throughput metrics (jobs/min)
type QueueThroughput struct {
	In  float64 `json:"in"`
	Out float64 `json:"out"`
}

// QueueSnapshot represents a point-in-time queue state
type QueueSnapshot struct {
	Name       string           `json:"name"`
	Driver     QueueDriver      `json:"driver"`
	Size       QueueSize        `json:"size"`
	Throughput *QueueThroughput `json:"throughput,omitempty"`
}

// CPUMetrics contains CPU usage data
type CPUMetrics struct {
	System  float64 `json:"system"`  // System-wide CPU % (0-100)
	Process float64 `json:"process"` // This process CPU % (0-100)
	Cores   int     `json:"cores"`   // Number of CPU cores
}

// SystemMemory contains system-wide memory metrics
type SystemMemory struct {
	Total uint64 `json:"total"` // Total bytes
	Free  uint64 `json:"free"`  // Free bytes
	Used  uint64 `json:"used"`  // Used bytes
}

// ProcessMemory contains process-specific memory metrics
type ProcessMemory struct {
	RSS       uint64 `json:"rss"`       // Resident Set Size
	HeapTotal uint64 `json:"heapTotal"` // Not applicable for Go, use RSS
	HeapUsed  uint64 `json:"heapUsed"`  // Not applicable for Go, use RSS
}

// MemoryMetrics contains both system and process memory
type MemoryMetrics struct {
	System  SystemMemory  `json:"system"`
	Process ProcessMemory `json:"process"`
}

// RuntimeInfo contains runtime metadata
type RuntimeInfo struct {
	Uptime    float64  `json:"uptime"`
	Framework string   `json:"framework"`
	Status    string   `json:"status"`            // "online", "degraded", "error"
	Errors    []string `json:"errors,omitempty"` // Connection errors or probe failures
}

// HeartbeatPayload is the complete payload sent to Zenith
type HeartbeatPayload struct {
	ID        string          `json:"id"`
	Service   string          `json:"service"`
	Language  Language        `json:"language"`
	Version   string          `json:"version"`
	PID       int             `json:"pid"`
	Hostname  string          `json:"hostname"`
	Platform  string          `json:"platform"`
	CPU       CPUMetrics      `json:"cpu"`
	Memory    MemoryMetrics   `json:"memory"`
	Queues    []QueueSnapshot `json:"queues,omitempty"`
	Runtime   RuntimeInfo            `json:"runtime"`
	Meta      map[string]interface{} `json:"meta,omitempty"` // Extra metadata like Laravel root, worker count
	Timestamp int64                  `json:"timestamp"`
}

// ============================================
// Remote Control Types (Phase 3)
// ============================================

// CommandType represents allowed command types
type CommandType string

const (
	CmdRetryJob      CommandType = "RETRY_JOB"
	CmdDeleteJob     CommandType = "DELETE_JOB"
	CmdLaravelAction CommandType = "LARAVEL_ACTION"
)

// AllowedCommands is the security allowlist
var AllowedCommands = []CommandType{CmdRetryJob, CmdDeleteJob, CmdLaravelAction}

// IsAllowed checks if a command type is in the allowlist
func (c CommandType) IsAllowed() bool {
	for _, allowed := range AllowedCommands {
		if c == allowed {
			return true
		}
	}
	return false
}

// CommandPayload contains command-specific data
type CommandPayload struct {
	Queue  string      `json:"queue,omitempty"`
	JobID  string      `json:"jobId,omitempty"`
	JobKey string      `json:"jobKey,omitempty"`
	Driver QueueDriver `json:"driver,omitempty"`
	Action string      `json:"action,omitempty"` // For LARAVEL_ACTION
}

// QuasarCommand represents a command from Zenith
type QuasarCommand struct {
	ID           string         `json:"id"`
	Type         CommandType    `json:"type"`
	TargetNodeID string         `json:"targetNodeId"`
	Payload      CommandPayload `json:"payload"`
	Timestamp    int64          `json:"timestamp"`
	Issuer       string         `json:"issuer"`
}

// CommandStatus represents execution result status
type CommandStatus string

const (
	StatusSuccess    CommandStatus = "success"
	StatusFailed     CommandStatus = "failed"
	StatusNotAllowed CommandStatus = "not_allowed"
)

// CommandResult represents the result of command execution
type CommandResult struct {
	CommandID string        `json:"commandId"`
	Status    CommandStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// NewSuccessResult creates a success result
func NewSuccessResult(commandID, message string) CommandResult {
	return CommandResult{
		CommandID: commandID,
		Status:    StatusSuccess,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewFailedResult creates a failed result
func NewFailedResult(commandID, message string) CommandResult {
	return CommandResult{
		CommandID: commandID,
		Status:    StatusFailed,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
}
