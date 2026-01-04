package types

import (
	"testing"
	"time"
)

func TestCommandTypeIsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		cmdType  CommandType
		expected bool
	}{
		{"RETRY_JOB allowed", CmdRetryJob, true},
		{"DELETE_JOB allowed", CmdDeleteJob, true},
		{"unknown command not allowed", CommandType("UNKNOWN"), false},
		{"empty command not allowed", CommandType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmdType.IsAllowed()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewSuccessResult(t *testing.T) {
	cmdID := "test-cmd-123"
	message := "Operation completed"

	result := NewSuccessResult(cmdID, message)

	if result.CommandID != cmdID {
		t.Errorf("Expected CommandID %s, got %s", cmdID, result.CommandID)
	}

	if result.Status != StatusSuccess {
		t.Errorf("Expected status %s, got %s", StatusSuccess, result.Status)
	}

	if result.Message != message {
		t.Errorf("Expected message %s, got %s", message, result.Message)
	}

	// Timestamp should be recent
	now := time.Now().UnixMilli()
	if result.Timestamp < now-1000 || result.Timestamp > now+1000 {
		t.Errorf("Timestamp %d is not recent (now: %d)", result.Timestamp, now)
	}
}

func TestNewFailedResult(t *testing.T) {
	cmdID := "test-cmd-456"
	message := "Operation failed"

	result := NewFailedResult(cmdID, message)

	if result.CommandID != cmdID {
		t.Errorf("Expected CommandID %s, got %s", cmdID, result.CommandID)
	}

	if result.Status != StatusFailed {
		t.Errorf("Expected status %s, got %s", StatusFailed, result.Status)
	}

	if result.Message != message {
		t.Errorf("Expected message %s, got %s", message, result.Message)
	}
}
