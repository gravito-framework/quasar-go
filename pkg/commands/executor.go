// Package commands provides command executor implementations.
package commands

import (
	"context"

	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// Executor handles execution of a specific command type
type Executor interface {
	// SupportedType returns the command type this executor handles
	SupportedType() types.CommandType

	// Execute runs the command and returns a result
	Execute(ctx context.Context, cmd *types.QuasarCommand, redis *redis.Client) types.CommandResult
}

// BaseExecutor provides common helper methods
type BaseExecutor struct{}

// Success creates a success result
func (e *BaseExecutor) Success(commandID, message string) types.CommandResult {
	return types.NewSuccessResult(commandID, message)
}

// Failed creates a failed result
func (e *BaseExecutor) Failed(commandID, message string) types.CommandResult {
	return types.NewFailedResult(commandID, message)
}
