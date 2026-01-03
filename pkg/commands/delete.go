package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// DeleteJobExecutor handles DELETE_JOB commands
type DeleteJobExecutor struct {
	BaseExecutor
}

// NewDeleteJobExecutor creates a new delete executor
func NewDeleteJobExecutor() *DeleteJobExecutor {
	return &DeleteJobExecutor{}
}

// SupportedType returns DELETE_JOB
func (e *DeleteJobExecutor) SupportedType() types.CommandType {
	return types.CmdDeleteJob
}

// Execute removes a job from the queue
func (e *DeleteJobExecutor) Execute(ctx context.Context, cmd *types.QuasarCommand, redisClient *redis.Client) types.CommandResult {
	queue := cmd.Payload.Queue
	jobKey := cmd.Payload.JobKey
	driver := cmd.Payload.Driver

	if queue == "" || jobKey == "" {
		return e.Failed(cmd.ID, "Missing queue or jobKey in payload")
	}

	switch driver {
	case types.DriverRedis:
		return e.deleteRedisJob(ctx, cmd.ID, redisClient, queue, jobKey)
	default:
		// Default to Laravel pattern
		return e.deleteLaravelJob(ctx, cmd.ID, redisClient, queue, jobKey)
	}
}

// deleteRedisJob removes job from {queue}:failed or {queue}
func (e *DeleteJobExecutor) deleteRedisJob(ctx context.Context, cmdID string, redisClient *redis.Client, queue, jobKey string) types.CommandResult {
	// Try failed queue first
	failedKey := queue + ":failed"
	removed, err := e.removeFromList(ctx, redisClient, failedKey, jobKey)
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to delete from failed queue: %v", err))
	}
	if removed {
		return e.Success(cmdID, fmt.Sprintf("Job deleted from %s", failedKey))
	}

	// Try waiting queue
	removed, err = e.removeFromList(ctx, redisClient, queue, jobKey)
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to delete from queue: %v", err))
	}
	if removed {
		return e.Success(cmdID, fmt.Sprintf("Job deleted from %s", queue))
	}

	return e.Failed(cmdID, "Job not found in any queue")
}

// deleteLaravelJob removes job from Laravel queue
func (e *DeleteJobExecutor) deleteLaravelJob(ctx context.Context, cmdID string, redisClient *redis.Client, queue, jobKey string) types.CommandResult {
	prefix := "queues"
	waitingKey := prefix + ":" + queue
	delayedKey := prefix + ":" + queue + ":delayed"
	reservedKey := prefix + ":" + queue + ":reserved"

	// Try waiting queue (List)
	removed, err := e.removeFromList(ctx, redisClient, waitingKey, jobKey)
	if err == nil && removed {
		return e.Success(cmdID, fmt.Sprintf("Job deleted from %s", waitingKey))
	}

	// Try delayed queue (ZSet)
	removed, err = e.removeFromZSet(ctx, redisClient, delayedKey, jobKey)
	if err == nil && removed {
		return e.Success(cmdID, fmt.Sprintf("Job deleted from %s", delayedKey))
	}

	// Try reserved queue (ZSet)
	removed, err = e.removeFromZSet(ctx, redisClient, reservedKey, jobKey)
	if err == nil && removed {
		return e.Success(cmdID, fmt.Sprintf("Job deleted from %s", reservedKey))
	}

	return e.Failed(cmdID, "Job not found in Laravel queues")
}

func (e *DeleteJobExecutor) removeFromList(ctx context.Context, redisClient *redis.Client, key, jobKey string) (bool, error) {
	// Get all items
	items, err := redisClient.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return false, err
	}

	// Find and remove
	for _, item := range items {
		if strings.Contains(item, jobKey) || item == jobKey {
			count, err := redisClient.LRem(ctx, key, 1, item).Result()
			if err != nil {
				return false, err
			}
			return count > 0, nil
		}
	}

	return false, nil
}

func (e *DeleteJobExecutor) removeFromZSet(ctx context.Context, redisClient *redis.Client, key, jobKey string) (bool, error) {
	// Get all members
	members, err := redisClient.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return false, err
	}

	// Find and remove
	for _, member := range members {
		if strings.Contains(member, jobKey) || member == jobKey {
			count, err := redisClient.ZRem(ctx, key, member).Result()
			if err != nil {
				return false, err
			}
			return count > 0, nil
		}
	}

	return false, nil
}

// Ensure DeleteJobExecutor implements Executor
var _ Executor = (*DeleteJobExecutor)(nil)
