package snapshotmanager

import (
	"context"
	"fmt"

	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/application/domain"
)

const (
	downloadContainerPrefix = "snapshot-download-"
)

// SnapshotManagerAdapter is a composite adapter that combines snapshots and docker adapters
// to provide high-level snapshot management operations
type SnapshotManagerAdapter struct {
	snapshots *snapshots.SnapshotsAdapter
	docker    *docker.DockerAdapter
}

// NewSnapshotManagerAdapter creates a new SnapshotManagerAdapter
func NewSnapshotManagerAdapter(
	snapshotsAdapter *snapshots.SnapshotsAdapter,
	dockerAdapter *docker.DockerAdapter,
) *SnapshotManagerAdapter {
	return &SnapshotManagerAdapter{
		snapshots: snapshotsAdapter,
		docker:    dockerAdapter,
	}
}

// DownloadAndMountSnapshot performs the complete snapshot download and mount process:
// 1. Stops the client container (if running)
// 2. Downloads and extracts the snapshot to the volume
func (s *SnapshotManagerAdapter) DownloadAndMountSnapshot(ctx context.Context, network string, client domain.ExecutionClientInfo) error {
	// 1. Stop the container (if running) - ignore errors as container might not exist
	_ = s.docker.StopContainer(ctx, client.ContainerName)

	// 2. Download and extract snapshot to volume using Docker container
	containerName := fmt.Sprintf("%s%s", downloadContainerPrefix, client.ShortName)
	if err := s.docker.RunSnapshotDownload(ctx, containerName, client.ShortName, network, client.VolumeTargetPath, s.snapshots.GetBaseURL()); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	return nil
}

// GetLatestBlockNumber fetches the latest available block number for a client
func (s *SnapshotManagerAdapter) GetLatestBlockNumber(ctx context.Context, network, client string) (string, error) {
	return s.snapshots.GetLatestBlockNumber(ctx, network, client)
}

// StopDownload stops the snapshot download container for a specific client
func (s *SnapshotManagerAdapter) StopDownload(ctx context.Context, clientShortName string) {
	containerName := fmt.Sprintf("%s%s", downloadContainerPrefix, clientShortName)
	_ = s.docker.StopContainerWithTimeout(ctx, containerName, 5)
}
