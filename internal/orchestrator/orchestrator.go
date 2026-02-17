package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

// Orchestrator manages multiple parallel agents, each in its own
// git worktree and Docker container.
type Orchestrator struct {
	engine   *sandbox.Engine
	auditor  *audit.Logger
	sessions map[string]*types.Session
	mu       sync.RWMutex
}

// New creates a new multi-agent orchestrator.
func New(engine *sandbox.Engine, auditor *audit.Logger) *Orchestrator {
	return &Orchestrator{
		engine:   engine,
		auditor:  auditor,
		sessions: make(map[string]*types.Session),
	}
}

// SpawnAgent creates a new agent with its own git worktree and sandbox container.
func (o *Orchestrator) SpawnAgent(ctx context.Context, projectDir string, agentName string, cfg types.SandboxConfig) (*types.Session, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.sessions[agentName]; exists {
		return nil, fmt.Errorf("agent %q already exists", agentName)
	}

	// Ensure absolute path for Docker mounts
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project dir: %w", err)
	}
	projectDir = absProjectDir

	// Create git worktree for isolation
	worktreeDir, err := createWorktree(projectDir, agentName)
	if err != nil {
		return nil, fmt.Errorf("creating worktree for %s: %w", agentName, err)
	}

	// Create sandbox with worktree as project dir
	session, err := o.engine.CreateSession(ctx, worktreeDir, cfg, agentName, nil)
	if err != nil {
		_ = removeWorktree(projectDir, worktreeDir)
		return nil, fmt.Errorf("creating sandbox for %s: %w", agentName, err)
	}

	session.WorktreeDir = worktreeDir
	o.sessions[agentName] = session

	return session, nil
}

// StopAgent stops an agent and cleans up its worktree.
func (o *Orchestrator) StopAgent(ctx context.Context, agentName string, merge bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	session, ok := o.sessions[agentName]
	if !ok {
		// Try to find the session from running Docker containers
		sessions, err := o.engine.ListSessions(ctx)
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}
		for _, s := range sessions {
			if s.AgentName == agentName {
				session = s
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("agent %q not found", agentName)
		}
	}

	// Stop sandbox
	if err := o.engine.StopSession(ctx, session); err != nil {
		return fmt.Errorf("stopping sandbox: %w", err)
	}

	// Merge changes if requested
	if merge && session.WorktreeDir != "" {
		if err := mergeWorktree(session.ProjectDir, agentName); err != nil {
			return fmt.Errorf("merging worktree: %w", err)
		}
	}

	// Clean up worktree
	if session.WorktreeDir != "" {
		_ = removeWorktree(session.ProjectDir, session.WorktreeDir)
	}

	delete(o.sessions, agentName)
	return nil
}

// ListAgents returns all active agent sessions.
func (o *Orchestrator) ListAgents() map[string]*types.Session {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]*types.Session, len(o.sessions))
	for k, v := range o.sessions {
		result[k] = v
	}
	return result
}

func createWorktree(projectDir, agentName string) (string, error) {
	branchName := "claudeshield/" + agentName
	worktreeDir := filepath.Join(projectDir, ".claudeshield", "worktrees", agentName)

	// Prune stale worktrees first
	pruneCmd := exec.Command("git", "-C", projectDir, "worktree", "prune")
	_ = pruneCmd.Run()

	// Remove leftover directory if it exists
	_ = os.RemoveAll(worktreeDir)

	// Delete branch if it already exists (from previous run)
	delCmd := exec.Command("git", "-C", projectDir, "branch", "-D", branchName)
	_ = delCmd.Run()

	// Create branch
	cmd := exec.Command("git", "-C", projectDir, "branch", branchName)
	_ = cmd.Run() // ignore if branch already exists

	// Create worktree
	cmd = exec.Command("git", "-C", projectDir, "worktree", "add", worktreeDir, branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %s: %w", string(out), err)
	}

	return worktreeDir, nil
}

func removeWorktree(projectDir, worktreeDir string) error {
	cmd := exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", string(out), err)
	}
	return nil
}

func mergeWorktree(projectDir, agentName string) error {
	branchName := "claudeshield/" + agentName

	// Commit any uncommitted changes in worktree
	worktreeDir := filepath.Join(projectDir, ".claudeshield", "worktrees", agentName)
	addCmd := exec.Command("git", "-C", worktreeDir, "add", "-A")
	_ = addCmd.Run()

	commitCmd := exec.Command("git", "-C", worktreeDir, "commit", "-m",
		fmt.Sprintf("ClaudeShield: agent %s changes", agentName))
	_ = commitCmd.Run()

	// Merge branch back into main
	mergeCmd := exec.Command("git", "-C", projectDir, "merge", "--no-ff",
		"-m", fmt.Sprintf("Merge agent %s", agentName),
		branchName)
	if out, err := mergeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("merge failed (resolve conflicts manually): %s: %w", string(out), err)
	}

	// Clean up branch
	delCmd := exec.Command("git", "-C", projectDir, "branch", "-d", branchName)
	_ = delCmd.Run()

	return nil
}
