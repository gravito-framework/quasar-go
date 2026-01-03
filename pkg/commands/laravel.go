package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gravito-framework/quasar-go/pkg/types"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/process"
)

// Ensure LaravelActionExecutor implements Executor
var _ Executor = (*LaravelActionExecutor)(nil)

// LaravelActionExecutor handles LARAVEL_ACTION commands
type LaravelActionExecutor struct {
	BaseExecutor
}

// NewLaravelActionExecutor creates a new Laravel executor
func NewLaravelActionExecutor() *LaravelActionExecutor {
	return &LaravelActionExecutor{}
}

// SupportedType returns LARAVEL_ACTION
func (e *LaravelActionExecutor) SupportedType() types.CommandType {
	return types.CmdLaravelAction
}

// Execute performs Laravel-specific operations like retry all or restart
func (e *LaravelActionExecutor) Execute(ctx context.Context, cmd *types.QuasarCommand, redisClient *redis.Client) types.CommandResult {
	action := cmd.Payload.Action
	if action == "" {
		return e.Failed(cmd.ID, "Missing action in payload")
	}

	// 1. Discover Laravel Root
	root, err := e.discoverLaravelRoot()
	if err != nil {
		return e.Failed(cmd.ID, fmt.Sprintf("Laravel project not found: %v", err))
	}

	// 2. Execute Action
	switch action {
	case "retry-all":
		return e.runArtisan(cmd.ID, root, "queue:retry", "all")
	case "restart":
		// Precision Targeting: Only restart workers on this machine for this project
		return e.restartLocalWorkers(cmd.ID, root)
	case "retry":
		jobID := cmd.Payload.JobID
		if jobID == "" {
			return e.Failed(cmd.ID, "Missing jobId for retry action")
		}
		return e.runArtisan(cmd.ID, root, "queue:retry", jobID)
	default:
		return e.Failed(cmd.ID, fmt.Sprintf("Unknown Laravel action: %s", action))
	}
}

// discoverLaravelRoot scans processes to find a running Laravel artisan command and its CWD
func (e *LaravelActionExecutor) discoverLaravelRoot() (string, error) {
	procs, err := process.Processes()
	if err != nil {
		return "", err
	}

	for _, p := range procs {
		if e.isLaravelWorker(p) {
			cwd, err := p.Cwd()
			if err == nil && cwd != "" {
				return cwd, nil
			}
		}
	}

	return "", fmt.Errorf("could not find any running artisan processes")
}

// restartLocalWorkers finds all local worker processes for the discovered root and sends SIGTERM
// This is "Supervisor Friendly" because Supervisor will see the exit and restart them.
// It is "Non-Interference" because it doesn't touch Redis/Cache to affect other servers.
func (e *LaravelActionExecutor) restartLocalWorkers(cmdID, root string) types.CommandResult {
	procs, err := process.Processes()
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Failed to list processes: %v", err))
	}

	count := 0
	var errors []string

	for _, p := range procs {
		// 1. Check if it's a Laravel worker
		if !e.isLaravelWorker(p) {
			continue
		}

		// 2. Check if it belongs to THIS project (Precision Targeting)
		cwd, err := p.Cwd()
		if err != nil {
			continue
		}

		if cwd == root {
			// 3. Send Signal (Supervisor Friendly)
			// Terminate sends SIGTERM, which tells Laravel workers to finish current job and exit.
			if err := p.Terminate(); err != nil {
				errors = append(errors, fmt.Sprintf("PID %d: %v", p.Pid, err))
			} else {
				count++
			}
		}
	}

	if count == 0 {
		return e.Failed(cmdID, fmt.Sprintf("No active workers found in %s to restart", root))
	}

	msg := fmt.Sprintf("Signaled %d local workers to restart", count)
	if len(errors) > 0 {
		msg += fmt.Sprintf(". Errors: %s", strings.Join(errors, "; "))
	}

	return e.Success(cmdID, msg)
}

// isLaravelWorker checks if a process looks like 'artisan queue:work' or 'horizon'
func (e *LaravelActionExecutor) isLaravelWorker(p *process.Process) bool {
	cmdline, err := p.Cmdline()
	if err != nil {
		return false
	}

	// Look for 'php artisan' or 'artisan'
	if !strings.Contains(cmdline, "artisan") {
		return false
	}

	// Look for worker indicators
	return strings.Contains(cmdline, "queue:work") || strings.Contains(cmdline, "horizon")
}

// runArtisan executes a php artisan command in the specified directory
func (e *LaravelActionExecutor) runArtisan(cmdID, root string, args ...string) types.CommandResult {
	// Security: Only allow specific artisan commands (whitelisting is indirect via Execute switch)
	allArgs := append([]string{"artisan"}, args...)
	cmd := exec.Command("php", allArgs...)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		return e.Failed(cmdID, fmt.Sprintf("Artisan command failed: %v\nOutput: %s", err, string(output)))
	}

	return e.Success(cmdID, fmt.Sprintf("Artisan success: %s", string(output)))
}
