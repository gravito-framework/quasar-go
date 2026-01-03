package queue

import (
	"context"

	"github.com/gravito-framework/quasar-go/pkg/probes"
	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// LaravelProbe monitors Laravel Queue with Redis driver
// Laravel uses specific key patterns:
//   - Waiting: queues:{name} (List)
//   - Delayed: queues:{name}:delayed (ZSet)
//   - Reserved (Active): queues:{name}:reserved (ZSet)
type LaravelProbe struct {
	client *redis.Client
	name   string
	prefix string
}

// NewLaravelProbe creates a probe for Laravel Queue
func NewLaravelProbe(client *redis.Client, queueName string) *LaravelProbe {
	return &LaravelProbe{
		client: client,
		name:   queueName,
		prefix: "queues", // Laravel default prefix
	}
}

// NewLaravelProbeWithPrefix creates a probe with custom prefix
func NewLaravelProbeWithPrefix(client *redis.Client, queueName, prefix string) *LaravelProbe {
	return &LaravelProbe{
		client: client,
		name:   queueName,
		prefix: prefix,
	}
}

// GetSnapshot returns current Laravel queue state
func (p *LaravelProbe) GetSnapshot() (*types.QueueSnapshot, error) {
	ctx := context.Background()

	// Laravel key patterns
	keyWaiting := p.prefix + ":" + p.name
	keyDelayed := p.prefix + ":" + p.name + ":delayed"
	keyReserved := p.prefix + ":" + p.name + ":reserved"

	// Use pipeline for efficiency
	pipe := p.client.Pipeline()
	waitingCmd := pipe.LLen(ctx, keyWaiting)
	delayedCmd := pipe.ZCard(ctx, keyDelayed)
	reservedCmd := pipe.ZCard(ctx, keyReserved)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// Note: Standard Laravel (without Horizon) stores failed jobs in Database (MySQL),
	// so we return 0 for failed unless using Horizon mode.
	return &types.QueueSnapshot{
		Name:   p.name,
		Driver: types.DriverRedis,
		Size: types.QueueSize{
			Waiting: waitingCmd.Val(),
			Active:  reservedCmd.Val(), // "reserved" in Laravel terms
			Failed:  0,                 // Cannot read from Redis easily in standard setup
			Delayed: delayedCmd.Val(),
		},
	}, nil
}

// Ensure LaravelProbe implements QueueProbe
var _ probes.QueueProbe = (*LaravelProbe)(nil)
