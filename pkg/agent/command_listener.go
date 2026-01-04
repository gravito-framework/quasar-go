package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gravito-framework/quasar-go/pkg/commands"
	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
)

// CommandListener subscribes to Redis Pub/Sub for incoming commands from Zenith.
type CommandListener struct {
	subscriber *redis.Client
	service    string
	nodeID     string
	logger     *slog.Logger
	executors  map[types.CommandType]commands.Executor
	isRunning  bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// NewCommandListener creates a new command listener
func NewCommandListener(
	subscriber *redis.Client,
	service string,
	nodeID string,
	logger *slog.Logger,
) *CommandListener {
	cl := &CommandListener{
		subscriber: subscriber,
		service:    service,
		nodeID:     nodeID,
		logger:     logger,
		executors:  make(map[types.CommandType]commands.Executor),
		stopChan:   make(chan struct{}),
	}

	// Register default executors
	cl.RegisterExecutor(commands.NewRetryJobExecutor())
	cl.RegisterExecutor(commands.NewDeleteJobExecutor())
	cl.RegisterExecutor(commands.NewLaravelActionExecutor())

	return cl
}

// RegisterExecutor registers a command executor
func (cl *CommandListener) RegisterExecutor(executor commands.Executor) {
	cl.executors[executor.SupportedType()] = executor
}

// channel returns the specific channel for this node
func (cl *CommandListener) channel() string {
	return fmt.Sprintf("gravito:quasar:cmd:%s:%s", cl.service, cl.nodeID)
}


// Start begins listening for commands
func (cl *CommandListener) Start(ctx context.Context, monitorRedis *redis.Client) error {
	cl.mu.Lock()
	if cl.isRunning {
		cl.mu.Unlock()
		return fmt.Errorf("command listener already running")
	}
	cl.isRunning = true
	cl.mu.Unlock()

	channel := cl.channel()

	// Subscribe to specific channel
	pubsub := cl.subscriber.Subscribe(ctx, channel)

	// Wait for confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	cl.logger.Info("📡 Listening for commands", "channel", channel)

	// Start message handler
	cl.wg.Add(1)
	go cl.handleMessages(ctx, pubsub, monitorRedis)

	return nil
}

// Stop stops the command listener
func (cl *CommandListener) Stop(ctx context.Context) error {
	cl.mu.Lock()
	if !cl.isRunning {
		cl.mu.Unlock()
		return nil
	}
	cl.isRunning = false
	cl.mu.Unlock()

	close(cl.stopChan)
	cl.wg.Wait()

	if err := cl.subscriber.Close(); err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}

	cl.logger.Info("CommandListener stopped")
	return nil
}

func (cl *CommandListener) handleMessages(ctx context.Context, pubsub *redis.PubSub, monitorRedis *redis.Client) {
	defer cl.wg.Done()
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-cl.stopChan:
			return
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			cl.processMessage(ctx, msg.Payload, monitorRedis)
		}
	}
}

func (cl *CommandListener) processMessage(ctx context.Context, payload string, monitorRedis *redis.Client) {
	var cmd types.QuasarCommand
	if err := json.Unmarshal([]byte(payload), &cmd); err != nil {
		cl.logger.Error("Failed to parse command", "error", err)
		return
	}

	cl.logger.Info("📥 Received command",
		"type", cmd.Type,
		"id", cmd.ID,
	)

	// Security check: Is this command type allowed?
	if !cmd.Type.IsAllowed() {
		cl.logger.Warn("⚠️ Command type not allowed", "type", cmd.Type)
		return
	}

	// Security check: Is this command for us?
	if cmd.TargetNodeID != cl.nodeID && cmd.TargetNodeID != "*" {
		cl.logger.Warn("⚠️ Command not for this node", "target", cmd.TargetNodeID)
		return
	}

	// Get executor
	executor, ok := cl.executors[cmd.Type]
	if !ok {
		cl.logger.Warn("⚠️ No executor for command type", "type", cmd.Type)
		return
	}

	// Execute
	result := executor.Execute(ctx, &cmd, monitorRedis)

	if result.Status == types.StatusSuccess {
		cl.logger.Info("✅ Command executed", "type", cmd.Type, "message", result.Message)
	} else {
		cl.logger.Error("❌ Command failed", "type", cmd.Type, "message", result.Message)
	}
}
