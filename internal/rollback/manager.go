package rollback

import (
	"context"
	"fmt"
	"time"

	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Manager handles checkpoint creation and rollback via Docker commit/layers.
type Manager struct {
	client      *client.Client
	checkpoints map[string][]types.Checkpoint
}

// New creates a new rollback manager.
func New(cli *client.Client) *Manager {
	return &Manager{
		client:      cli,
		checkpoints: make(map[string][]types.Checkpoint),
	}
}

// CreateCheckpoint creates a snapshot of the container state.
func (m *Manager) CreateCheckpoint(ctx context.Context, session *types.Session, description string) (*types.Checkpoint, error) {
	cpID := fmt.Sprintf("cs-cp-%d", time.Now().UnixMilli())

	commitResp, err := m.client.ContainerCommit(ctx, session.ContainerID, container.CommitOptions{
		Reference: fmt.Sprintf("claudeshield/checkpoint:%s", cpID),
		Comment:   description,
		Pause:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating checkpoint: %w", err)
	}

	cp := types.Checkpoint{
		ID:          cpID,
		SessionID:   session.ID,
		ImageID:     commitResp.ID,
		Description: description,
		CreatedAt:   time.Now(),
	}

	m.checkpoints[session.ID] = append(m.checkpoints[session.ID], cp)
	return &cp, nil
}

// Rollback restores a container to a previous checkpoint.
func (m *Manager) Rollback(ctx context.Context, session *types.Session, checkpointID string) error {
	checkpoints, ok := m.checkpoints[session.ID]
	if !ok {
		return fmt.Errorf("no checkpoints for session %s", session.ID)
	}

	var target *types.Checkpoint
	for i, cp := range checkpoints {
		if cp.ID == checkpointID {
			target = &checkpoints[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("checkpoint %s not found", checkpointID)
	}

	// Stop current container
	timeout := 5
	stopOpts := container.StopOptions{Timeout: &timeout}
	_ = m.client.ContainerStop(ctx, session.ContainerID, stopOpts)
	_ = m.client.ContainerRemove(ctx, session.ContainerID, container.RemoveOptions{})

	// Start new container from checkpoint image
	resp, err := m.client.ContainerCreate(ctx, &container.Config{
		Image:      target.ImageID,
		WorkingDir: "/workspace",
		Tty:        true,
		OpenStdin:  true,
	}, nil, nil, nil, session.ID)
	if err != nil {
		return fmt.Errorf("creating container from checkpoint: %w", err)
	}

	if err := m.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting restored container: %w", err)
	}

	session.ContainerID = resp.ID
	session.UpdatedAt = time.Now()

	return nil
}

// ListCheckpoints returns all checkpoints for a session.
func (m *Manager) ListCheckpoints(sessionID string) []types.Checkpoint {
	return m.checkpoints[sessionID]
}

// RollbackToLatest rolls back to the most recent checkpoint.
func (m *Manager) RollbackToLatest(ctx context.Context, session *types.Session) error {
	checkpoints, ok := m.checkpoints[session.ID]
	if !ok || len(checkpoints) == 0 {
		return fmt.Errorf("no checkpoints for session %s", session.ID)
	}

	latest := checkpoints[len(checkpoints)-1]
	return m.Rollback(ctx, session, latest.ID)
}
