// Package queue provides queue probe implementations.
package queue

import (
	"context"

	"github.com/gravito-framework/quasar-go/pkg/probes"
	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// RedisListProbe monitors a simple Redis List queue
type RedisListProbe struct {
	client *redis.Client
	name   string
}

// NewRedisListProbe creates a probe for a Redis List queue
func NewRedisListProbe(client *redis.Client, queueName string) *RedisListProbe {
	return &RedisListProbe{
		client: client,
		name:   queueName,
	}
}

// GetSnapshot returns current queue state
func (p *RedisListProbe) GetSnapshot() (*types.QueueSnapshot, error) {
	ctx := context.Background()

	// Get waiting queue length
	waiting, err := p.client.LLen(ctx, p.name).Result()
	if err != nil {
		return nil, err
	}

	// Check for common patterns: {queue}:failed, {queue}:delayed
	failed, _ := p.client.LLen(ctx, p.name+":failed").Result()
	delayed, _ := p.client.LLen(ctx, p.name+":delayed").Result()
	active, _ := p.client.LLen(ctx, p.name+":active").Result()

	return &types.QueueSnapshot{
		Name:   p.name,
		Driver: types.DriverRedis,
		Size: types.QueueSize{
			Waiting: waiting,
			Active:  active,
			Failed:  failed,
			Delayed: delayed,
		},
	}, nil
}

// Ensure RedisListProbe implements QueueProbe
var _ probes.QueueProbe = (*RedisListProbe)(nil)
