package snapshotmanager

import (
	"context"
	"fmt"
	"sync"

	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
)

const (
	downloadContainerPrefix = "snapshot-download-"
)

var logPrefix = "SnapshotManager"

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
	exists, err := s.blockNumber.BlockNumberExists(ctx, client.VolumeTargetPath)
	if err != nil {
		return false, err
	}

	if !exists {
		logger.InfoWithPrefix(logPrefix, "No existing snapshot for %s, download needed", client.ShortName)
		return true, nil
	}

	// Check if newer snapshot is available
	isNewer, err := s.blockNumber.IsNewerSnapshot(ctx, client.VolumeTargetPath, latestBlockNumber)
	if err != nil {
		return false, err
	}

	if isNewer {
		currentBlock, _ := s.blockNumber.ReadBlockNumber(ctx, client.VolumeTargetPath)
		logger.InfoWithPrefix(logPrefix, "Newer snapshot available for %s: current=%s, latest=%s",
			client.ShortName, currentBlock, latestBlockNumber)
		return true, nil
	}

	return false, nil
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

	logger.InfoWithPrefix(logPrefix, "Starting snapshot download for %s (block %s)", client.ShortName, latestBlockNumber)

	// 1. Stop the container (if running)
	logger.InfoWithPrefix(logPrefix, "Stopping container %s", client.ContainerName)
	if err := s.docker.StopContainer(ctx, client.ContainerName); err != nil {
		// Log but don't fail - container might not exist or not be running
		logger.WarnWithPrefix(logPrefix, "Could not stop container %s: %v", client.ContainerName, err)
	}

	// 2. Download and extract snapshot to volume using Docker container
	containerName := fmt.Sprintf("%s%s", downloadContainerPrefix, client.ShortName)
	logger.InfoWithPrefix(logPrefix, "Downloading and extracting snapshot to %s", client.VolumeTargetPath)
	if err := s.docker.RunSnapshotDownload(ctx, containerName, client.ShortName, network, client.VolumeTargetPath, s.snapshots.GetBaseURL()); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	// 3. Write block number file
	if err := s.blockNumber.WriteBlockNumber(ctx, client.VolumeTargetPath, latestBlockNumber); err != nil {
		return fmt.Errorf("failed to write block number: %w", err)
	}

	logger.InfoWithPrefix(logPrefix, "Successfully downloaded and mounted snapshot for %s (block %s)",
		client.ShortName, latestBlockNumber)

	return nil
}

// GetLatestBlockNumber fetches the latest available block number for a client
func (s *SnapshotManagerAdapter) GetLatestBlockNumber(ctx context.Context, network, client string) (string, error) {
	return s.snapshots.GetLatestBlockNumber(ctx, network, client)
}

// StopAllDownloads stops all running snapshot download containers in parallel
func (s *SnapshotManagerAdapter) StopAllDownloads(ctx context.Context) {
	logger.InfoWithPrefix(logPrefix, "Stopping all snapshot download containers...")

	containerIDs, err := s.docker.ListContainersByPrefix(ctx, downloadContainerPrefix)
	if err != nil {
		logger.WarnWithPrefix(logPrefix, "Failed to list download containers: %v", err)
		return
	}

	if len(containerIDs) == 0 {
		logger.DebugWithPrefix(logPrefix, "No download containers running")
		return
	}

	// Stop all containers in parallel
	var wg sync.WaitGroup
	for _, id := range containerIDs {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			logger.InfoWithPrefix(logPrefix, "Stopping download container %s", containerID)
			if err := s.docker.StopContainerWithTimeout(ctx, containerID, 5); err != nil {
				logger.WarnWithPrefix(logPrefix, "Failed to stop container %s: %v", containerID, err)
			}
		}(id)
	}
	wg.Wait()

	logger.InfoWithPrefix(logPrefix, "All download containers stopped")
}
