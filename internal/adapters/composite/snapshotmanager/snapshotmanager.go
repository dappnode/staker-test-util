package snapshotmanager

import (
	"context"
	"fmt"
	"sync"

	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/application/domain"
)

const (
	downloadContainerPrefix = "snapshot-download-"
)

// SnapshotManagerAdapter is a composite adapter that combines snapshots, docker, and blocknumber adapters
// to provide high-level snapshot management operations
type SnapshotManagerAdapter struct {
	snapshots   *snapshots.SnapshotsAdapter
	docker      *docker.DockerAdapter
	blockNumber *blocknumber.BlockNumberAdapter
}

// NewSnapshotManagerAdapter creates a new SnapshotManagerAdapter
func NewSnapshotManagerAdapter(
	snapshotsAdapter *snapshots.SnapshotsAdapter,
	dockerAdapter *docker.DockerAdapter,
	blockNumberAdapter *blocknumber.BlockNumberAdapter,
) *SnapshotManagerAdapter {
	return &SnapshotManagerAdapter{
		snapshots:   snapshotsAdapter,
		docker:      dockerAdapter,
		blockNumber: blockNumberAdapter,
	}
}

// NeedsSnapshotDownload determines if a snapshot needs to be downloaded for the given client
// Returns true if no snapshot exists or if a newer snapshot is available
func (s *SnapshotManagerAdapter) NeedsSnapshotDownload(ctx context.Context, client domain.ExecutionClientInfo, latestBlockNumber string) (bool, error) {
	// Check if block number file exists
	exists, err := s.blockNumber.BlockNumberExists(ctx)
	if err != nil {
		return false, err
	}

	if !exists {
		return true, nil
	}

	// Check if newer snapshot is available
	isNewer, err := s.blockNumber.IsNewerSnapshot(ctx, latestBlockNumber)
	if err != nil {
		return false, err
	}

	return isNewer, nil
}

// DownloadAndMountSnapshot performs the complete snapshot download and mount process:
// 1. Stops the client container (if running)
// 2. Downloads and extracts the snapshot to the volume
// 3. Writes the block number file
func (s *SnapshotManagerAdapter) DownloadAndMountSnapshot(ctx context.Context, network string, client domain.ExecutionClientInfo) error {
	// Get latest block number first
	latestBlockNumber, err := s.snapshots.GetLatestBlockNumber(ctx, network, client.ShortName)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}

	// 1. Stop the container (if running) - ignore errors as container might not exist
	_ = s.docker.StopContainer(ctx, client.ContainerName)

	// 2. Download and extract snapshot to volume using Docker container
	containerName := fmt.Sprintf("%s%s", downloadContainerPrefix, client.ShortName)
	if err := s.docker.RunSnapshotDownload(ctx, containerName, client.ShortName, network, client.VolumeTargetPath, s.snapshots.GetBaseURL()); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	// 3. Write block number file
	if err := s.blockNumber.WriteBlockNumber(ctx, latestBlockNumber); err != nil {
		return fmt.Errorf("failed to write block number: %w", err)
	}

	return nil
}

// GetLatestBlockNumber fetches the latest available block number for a client
func (s *SnapshotManagerAdapter) GetLatestBlockNumber(ctx context.Context, network, client string) (string, error) {
	return s.snapshots.GetLatestBlockNumber(ctx, network, client)
}

// StopAllDownloads stops all running snapshot download containers in parallel
func (s *SnapshotManagerAdapter) StopAllDownloads(ctx context.Context) {
	containerIDs, err := s.docker.ListContainersByPrefix(ctx, downloadContainerPrefix)
	if err != nil {
		return
	}

	if len(containerIDs) == 0 {
		return
	}

	// Stop all containers in parallel
	var wg sync.WaitGroup
	for _, id := range containerIDs {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			_ = s.docker.StopContainerWithTimeout(ctx, containerID, 5)
		}(id)
	}
	wg.Wait()
}
