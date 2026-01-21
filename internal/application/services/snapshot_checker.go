package services

import (
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/adapters/shared/progress"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"sync"
	"time"
)

var snapshotLogPrefix = "SnapshotChecker"

// SnapshotCheckerService handles snapshot checking and downloading
type SnapshotCheckerService struct {
	snapshots   *snapshots.SnapshotsAdapter
	docker      *docker.DockerAdapter
	progress    *progress.ProgressAdapter
	blockNumber *blocknumber.BlockNumberAdapter
	config      domain.SnapshotCheckerConfig
}

// NewSnapshotCheckerService creates a new SnapshotCheckerService
func NewSnapshotCheckerService(
	snapshotsAdapter *snapshots.SnapshotsAdapter,
	dockerAdapter *docker.DockerAdapter,
	progressAdapter *progress.ProgressAdapter,
	blockNumberAdapter *blocknumber.BlockNumberAdapter,
	config domain.SnapshotCheckerConfig,
) *SnapshotCheckerService {
	return &SnapshotCheckerService{
		snapshots:   snapshotsAdapter,
		docker:      dockerAdapter,
		progress:    progressAdapter,
		blockNumber: blockNumberAdapter,
		config:      config,
	}
}

// Start starts the snapshot checker cron job
func (s *SnapshotCheckerService) Start(ctx context.Context, runOnce bool) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "Starting snapshot checker for network: %s", s.config.Network)
	logger.InfoWithPrefix(snapshotLogPrefix, "Managing %d execution clients", len(s.config.ExecutionClients))

	// Run immediately on startup
	if err := s.CheckAndUpdateSnapshots(ctx); err != nil {
		logger.ErrorWithPrefix(snapshotLogPrefix, "Initial snapshot check failed: %v", err)
		// Return error on initial failure to avoid waiting the cron interval
		return err
	}

	if runOnce {
		logger.InfoWithPrefix(snapshotLogPrefix, "Run-once mode, exiting after initial check")
		return nil
	}

	// Start cron loop
	ticker := time.NewTicker(time.Duration(s.config.CronIntervalSec) * time.Second)
	defer ticker.Stop()

	logger.InfoWithPrefix(snapshotLogPrefix, "Cron job scheduled every %d seconds (%s)",
		s.config.CronIntervalSec, time.Duration(s.config.CronIntervalSec)*time.Second)

	for {
		select {
		case <-ctx.Done():
			logger.InfoWithPrefix(snapshotLogPrefix, "Snapshot checker stopped: %v", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			logger.InfoWithPrefix(snapshotLogPrefix, "Running scheduled snapshot check")
			if err := s.CheckAndUpdateSnapshots(ctx); err != nil {
				logger.ErrorWithPrefix(snapshotLogPrefix, "Scheduled snapshot check failed: %v", err)
				// Continue running even on failure
			}
		}
	}
}

// CheckAndUpdateSnapshots checks all configured clients and updates snapshots as needed
func (s *SnapshotCheckerService) CheckAndUpdateSnapshots(ctx context.Context) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "Checking snapshots for %d clients", len(s.config.ExecutionClients))

	// Check if a download is already in progress
	inProgress, err := s.progress.IsDownloadInProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to check download progress: %w", err)
	}
	if inProgress {
		logger.InfoWithPrefix(snapshotLogPrefix, "Download already in progress, skipping this cycle")
		return nil
	}

	// Set download in progress for this cycle
	if err := s.progress.SetDownloadInProgress(ctx); err != nil {
		return fmt.Errorf("failed to set download in progress: %w", err)
	}

	// Ensure we clear the progress file on completion (success or failure)
	defer func() {
		if err := s.progress.ClearDownloadInProgress(ctx); err != nil {
			logger.ErrorWithPrefix(snapshotLogPrefix, "Failed to clear download in progress: %v", err)
		} else {
			logger.InfoWithPrefix(snapshotLogPrefix, "Cleared download in progress file")
		}
	}()

	// Process clients in parallel using goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, len(s.config.ExecutionClients))

	for _, client := range s.config.ExecutionClients {
		wg.Add(1)
		go func(client domain.ExecutionClientInfo) {
			defer wg.Done()
			if err := s.checkAndUpdateClient(ctx, client); err != nil {
				errChan <- fmt.Errorf("client %s: %w", client.ShortName, err)
			}
		}(client)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		logger.ErrorWithPrefix(snapshotLogPrefix, "Snapshot check completed with %d errors", len(errors))
		return fmt.Errorf("snapshot check had %d errors: %v", len(errors), errors)
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "Snapshot check completed successfully")
	return nil
}

// checkAndUpdateClient checks a single client and updates snapshot if needed
func (s *SnapshotCheckerService) checkAndUpdateClient(ctx context.Context, client domain.ExecutionClientInfo) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "Checking client: %s (%s)", client.ShortName, client.DnpName)

	// Get latest available block number from ethpandaops
	latestBlockNumber, err := s.snapshots.GetLatestBlockNumber(ctx, s.config.Network, client.ShortName)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	logger.InfoWithPrefix(snapshotLogPrefix, "Latest available snapshot for %s: block %s", client.ShortName, latestBlockNumber)

	// Check if we need to download
	needsDownload, err := s.needsSnapshotDownload(ctx, client, latestBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to check if snapshot needed: %w", err)
	}

	if !needsDownload {
		logger.InfoWithPrefix(snapshotLogPrefix, "Snapshot for %s is up to date, skipping", client.ShortName)
		return nil
	}

	// Download and mount snapshot
	if err := s.downloadAndMountSnapshot(ctx, client, latestBlockNumber); err != nil {
		return fmt.Errorf("failed to download and mount snapshot: %w", err)
	}

	return nil
}

// needsSnapshotDownload determines if a snapshot needs to be downloaded
func (s *SnapshotCheckerService) needsSnapshotDownload(ctx context.Context, client domain.ExecutionClientInfo, latestBlockNumber string) (bool, error) {
	// Check if block number file exists
	exists, err := s.blockNumber.BlockNumberExists(ctx, client.VolumeTargetPath)
	if err != nil {
		return false, err
	}

	if !exists {
		logger.InfoWithPrefix(snapshotLogPrefix, "No existing snapshot for %s, download needed", client.ShortName)
		return true, nil
	}

	// Check if newer snapshot is available
	isNewer, err := s.blockNumber.IsNewerSnapshot(ctx, client.VolumeTargetPath, latestBlockNumber)
	if err != nil {
		return false, err
	}

	if isNewer {
		currentBlock, _ := s.blockNumber.ReadBlockNumber(ctx, client.VolumeTargetPath)
		logger.InfoWithPrefix(snapshotLogPrefix, "Newer snapshot available for %s: current=%s, latest=%s",
			client.ShortName, currentBlock, latestBlockNumber)
		return true, nil
	}

	return false, nil
}

// downloadAndMountSnapshot performs the complete snapshot download and mount process
func (s *SnapshotCheckerService) downloadAndMountSnapshot(ctx context.Context, client domain.ExecutionClientInfo, blockNumber string) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "Starting snapshot download for %s (block %s)", client.ShortName, blockNumber)

	// 1. Stop the container (if running)
	logger.InfoWithPrefix(snapshotLogPrefix, "Stopping container %s", client.ContainerName)
	if err := s.docker.StopContainer(ctx, client.ContainerName); err != nil {
		// Log but don't fail - container might not exist or not be running
		logger.WarnWithPrefix(snapshotLogPrefix, "Could not stop container %s: %v", client.ContainerName, err)
	}

	// 2. Download and extract snapshot to volume
	logger.InfoWithPrefix(snapshotLogPrefix, "Downloading and extracting snapshot to %s", client.VolumeTargetPath)
	if err := s.snapshots.DownloadAndExtract(ctx, s.config.Network, client.ShortName, client.VolumeTargetPath); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	// 3. Write block number file
	if err := s.blockNumber.WriteBlockNumber(ctx, client.VolumeTargetPath, blockNumber); err != nil {
		return fmt.Errorf("failed to write block number: %w", err)
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "Successfully downloaded and mounted snapshot for %s (block %s)",
		client.ShortName, blockNumber)

	return nil
}
