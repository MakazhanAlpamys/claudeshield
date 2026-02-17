package sandbox

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/policy"
	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const (
	SandboxImage   = "claudeshield/sandbox:latest"
	ContainerLabel = "claudeshield.managed"
)

// Engine manages Docker-based sandbox containers.
type Engine struct {
	client  *client.Client
	auditor *audit.Logger
	policy  *policy.Engine
}

// New creates a new sandbox engine connected to the local Docker daemon.
func New(auditor *audit.Logger, policyEngine *policy.Engine) (*Engine, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Docker is not running or not accessible: %w", err)
	}

	return &Engine{client: cli, auditor: auditor, policy: policyEngine}, nil
}

// Client returns the underlying Docker client (used by rollback manager).
func (e *Engine) Client() *client.Client {
	return e.client
}

// CreateSession creates and starts a new sandbox container for the project.
// If secrets are provided, they are injected as environment variables into the container.
func (e *Engine) CreateSession(ctx context.Context, projectDir string, cfg types.SandboxConfig, agentName string, secrets map[string]string) (*types.Session, error) {
	sessionID := fmt.Sprintf("cs-%s-%d", agentName, time.Now().UnixMilli())

	if err := e.ensureImage(ctx); err != nil {
		return nil, err
	}

	mounts, err := e.buildMounts(projectDir, cfg)
	if err != nil {
		return nil, err
	}

	hostCfg := &container.HostConfig{
		Mounts:     mounts,
		AutoRemove: false,
		SecurityOpt: []string{
			"no-new-privileges:true",
		},
		CapDrop: []string{"ALL"},
		CapAdd:  []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "SETGID", "SETUID"},
	}

	if cfg.MemoryLimit != "" {
		mem := parseMemoryLimit(cfg.MemoryLimit)
		if mem > 0 {
			hostCfg.Resources.Memory = mem
		}
	}
	if cfg.CPULimit > 0 {
		hostCfg.Resources.NanoCPUs = int64(cfg.CPULimit * 1e9)
	}

	if !cfg.Network {
		hostCfg.NetworkMode = "none"
	}

	if cfg.UseGVisor {
		hostCfg.Runtime = "runsc"
	}

	// Build environment variables from secrets
	var envVars []string
	for k, v := range secrets {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	containerCfg := &container.Config{
		Image: SandboxImage,
		Labels: map[string]string{
			ContainerLabel:           "true",
			"claudeshield.session":   sessionID,
			"claudeshield.agent":     agentName,
			"claudeshield.project":   projectDir,
		},
		Env:        envVars,
		WorkingDir: "/workspace",
		Tty:        true,
		OpenStdin:  true,
	}

	resp, err := e.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, sessionID)
	if err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}

	if err := e.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}

	session := &types.Session{
		ID:          sessionID,
		ProjectDir:  projectDir,
		ContainerID: resp.ID,
		State:       types.SessionRunning,
		AgentName:   agentName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if e.auditor != nil {
		e.auditor.Log(types.AuditEntry{
			Timestamp: time.Now(),
			SessionID: sessionID,
			AgentName: agentName,
			EventType: "session_created",
			Action:    types.ActionAllow,
		})
	}

	return session, nil
}

// StopSession stops and removes the sandbox container.
func (e *Engine) StopSession(ctx context.Context, session *types.Session) error {
	timeout := 10
	stopOpts := container.StopOptions{Timeout: &timeout}

	if err := e.client.ContainerStop(ctx, session.ContainerID, stopOpts); err != nil {
		return fmt.Errorf("stopping container: %w", err)
	}

	if err := e.client.ContainerRemove(ctx, session.ContainerID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("removing container: %w", err)
	}

	session.State = types.SessionStopped
	session.UpdatedAt = time.Now()

	if e.auditor != nil {
		e.auditor.Log(types.AuditEntry{
			Timestamp: time.Now(),
			SessionID: session.ID,
			AgentName: session.AgentName,
			EventType: "session_stopped",
			Action:    types.ActionAllow,
		})
	}

	return nil
}

// ExecCommand runs a command inside the sandbox container, after policy check.
func (e *Engine) ExecCommand(ctx context.Context, session *types.Session, cmd []string) (string, error) {
	// Policy check before execution
	commandStr := strings.Join(cmd, " ")
	if e.policy != nil {
		result := e.policy.EvaluateCommand(commandStr)

		if e.auditor != nil {
			entry := types.AuditEntry{
				Timestamp: time.Now(),
				SessionID: session.ID,
				AgentName: session.AgentName,
				EventType: "command_exec",
				Command:   commandStr,
				Action:    result.Action,
				Reason:    result.Reason,
			}
			if result.Rule != nil {
				entry.RulePattern = result.Rule.Pattern
			}
			e.auditor.Log(entry)
		}

		if !result.Allowed {
			return "", fmt.Errorf("policy blocked: %s (reason: %s)", commandStr, result.Reason)
		}
	}

	execCfg := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	}

	execResp, err := e.client.ContainerExecCreate(ctx, session.ContainerID, execCfg)
	if err != nil {
		return "", fmt.Errorf("creating exec: %w", err)
	}

	attachResp, err := e.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attaching to exec: %w", err)
	}
	defer attachResp.Close()

	output, err := io.ReadAll(attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("reading exec output: %w", err)
	}

	return string(output), nil
}

// ListSessions returns all active ClaudeShield containers.
func (e *Engine) ListSessions(ctx context.Context) ([]*types.Session, error) {
	containers, err := e.client.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	var sessions []*types.Session
	for _, c := range containers {
		if c.Labels[ContainerLabel] != "true" {
			continue
		}

		state := types.SessionRunning
		if c.State == "exited" {
			state = types.SessionStopped
		} else if c.State == "paused" {
			state = types.SessionPaused
		}

		sessions = append(sessions, &types.Session{
			ID:          c.Labels["claudeshield.session"],
			ContainerID: c.ID,
			AgentName:   c.Labels["claudeshield.agent"],
			ProjectDir:  c.Labels["claudeshield.project"],
			State:       state,
			CreatedAt:   time.Unix(c.Created, 0),
		})
	}

	return sessions, nil
}

// Close closes the Docker client.
func (e *Engine) Close() error {
	return e.client.Close()
}

func (e *Engine) ensureImage(ctx context.Context) error {
	images, err := e.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == SandboxImage {
				return nil
			}
		}
	}

	reader, err := e.client.ImagePull(ctx, SandboxImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling sandbox image %s: %w", SandboxImage, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)

	return nil
}

func (e *Engine) buildMounts(projectDir string, cfg types.SandboxConfig) ([]mount.Mount, error) {
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   projectDir,
			Target:   "/workspace",
			ReadOnly: false,
		},
	}

	for _, ro := range cfg.ReadOnly {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   ro,
			Target:   ro,
			ReadOnly: true,
		})
	}

	return mounts, nil
}

func parseMemoryLimit(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	multiplier := int64(1)

	if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	}

	var val int64
	fmt.Sscanf(s, "%d", &val)
	return val * multiplier
}
