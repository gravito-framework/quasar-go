package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// RetryJobExecutor handles RETRY_JOB commands
type RetryJobExecutor struct {
	BaseExecutor
}

// NewRetryJobExecutor creates a new retry executor
func NewRetryJobExecutor() *RetryJobExecutor {
	return &RetryJobExecutor{}
}

// SupportedType returns RETRY_JOB
func (e *RetryJobExecutor) SupportedType() types.CommandType {
	return types.CmdRetryJob
}

// Execute moves a failed job back to the waiting queue
func (e *RetryJobExecutor) Execute(ctx context.Context, cmd *types.QuasarCommand, redisClient *redis.Client) types.CommandResult {
	queue := cmd.Payload.Queue
	jobKey := cmd.Payload.JobKey
	driver := cmd.Payload.Driver

	if queue == "" || jobKey == "" {
		return e.Failed(cmd.ID, "Missing queue or jobKey in payload")
	}

	switch driver {
	case types.DriverRedis:
		return e.retryRedisJob(ctx, cmd.ID, redisClient, queue, jobKey)
	default:
		// Default to Laravel pattern
		return e.retryLaravelJob(ctx, cmd.ID, redisClient, queue, jobKey)
	}
}

// retryRedisJob moves job from {queue}:failed -> {queue}
func (e *RetryJobExecutor) retryRedisJob(ctx context.Context, cmdID string, redisClient *redis.Client, queue, jobKey string) types.CommandResult {
	failedKey := queue + ":failed"
	waitingKey := queue

	// Get all jobs from failed list
	jobs, err := redisClient.LRange(ctx, failedKey, 0, -1).Result()
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to read failed queue: %v", err))
	}

	// Find the job
	jobIndex := -1
	var foundJob string
	for i, job := range jobs {
		if strings.Contains(job, jobKey) || job == jobKey {
			jobIndex = i
			foundJob = job
			break
		}
	}

	if jobIndex == -1 {
		return e.Failed(cmdID, fmt.Sprintf("Job not found in %s", failedKey))
	}

	// Atomic move: LREM + RPUSH
	pipe := redisClient.TxPipeline()
	pipe.LRem(ctx, failedKey, 1, foundJob)
	pipe.RPush(ctx, waitingKey, foundJob)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to move job: %v", err))
	}

	return e.Success(cmdID, fmt.Sprintf("Job moved to %s", waitingKey))
}

// retryLaravelJob pushes job back to Laravel queue
func (e *RetryJobExecutor) retryLaravelJob(ctx context.Context, cmdID string, redisClient *redis.Client, queue, jobKey string) types.CommandResult {
	prefix := "queues"
	waitingKey := prefix + ":" + queue

	// Push the job back (jobKey should be the serialized job data)
	err := redisClient.RPush(ctx, waitingKey, jobKey).Err()
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to push job: %v", err))
	}

	return e.Success(cmdID, fmt.Sprintf("Job pushed to %s", waitingKey))
}

// Ensure RetryJobExecutor implements Executor
var _ Executor = (*RetryJobExecutor)(nil)
