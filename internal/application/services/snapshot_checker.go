package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"time"
)

var snapshotLogPrefix = "SnapshotChecker"

// SnapshotCheckerService handles snapshot checking and downloading
type SnapshotCheckerService struct {
	snapshotManager  ports.SnapshotManager
	downloadProgress ports.DownloadProgress
	testProgress     ports.TestProgress
	blockNumber      ports.BlockNumber
	config           domain.SnapshotCheckerConfig
}

// NewSnapshotCheckerService creates a new SnapshotCheckerService
func NewSnapshotCheckerService(
	snapshotManager ports.SnapshotManager,
	downloadProgress ports.DownloadProgress,
	testProgress ports.TestProgress,
	blockNumber ports.BlockNumber,
	config domain.SnapshotCheckerConfig,
) *SnapshotCheckerService {
	return &SnapshotCheckerService{
		snapshotManager:  snapshotManager,
		downloadProgress: downloadProgress,
		testProgress:     testProgress,
		blockNumber:      blockNumber,
		config:           config,
	}
}

// GetSnapshotManager returns the snapshot manager adapter (for shutdown handling)
func (s *SnapshotCheckerService) GetSnapshotManager() ports.SnapshotManager {
	return s.snapshotManager
}

// Start starts the snapshot checker cron job
func (s *SnapshotCheckerService) Start(ctx context.Context, runOnce bool) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "Starting snapshot checker for network: %s", s.config.Network)
	logger.InfoWithPrefix(snapshotLogPrefix, "Managing execution client: %s", s.config.ExecutionClient.ShortName)

	// Run immediately on startup
	logger.InfoWithPrefix(snapshotLogPrefix, "Running initial snapshot check...")
	if err := s.CheckAndUpdateSnapshots(ctx); err != nil {
		logger.ErrorWithPrefix(snapshotLogPrefix, "Initial snapshot check failed: %v", err)
		return err
	}

	if runOnce {
		logger.InfoWithPrefix(snapshotLogPrefix, "Run-once mode enabled, exiting after initial check")
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
			logger.InfoWithPrefix(snapshotLogPrefix, "Running scheduled snapshot check...")
			if err := s.CheckAndUpdateSnapshots(ctx); err != nil {
				logger.ErrorWithPrefix(snapshotLogPrefix, "Scheduled snapshot check failed: %v", err)
				// Continue running even on failure
			}
		}
	}
}

// CheckAndUpdateSnapshots checks the configured client and updates snapshot if needed
func (s *SnapshotCheckerService) CheckAndUpdateSnapshots(ctx context.Context) error {
	client := s.config.ExecutionClient
	logger.InfoWithPrefix(snapshotLogPrefix, "Checking snapshot for client: %s", client.ShortName)

	if err := s.checkAndUpdateClient(ctx, client); err != nil {
		return fmt.Errorf("client %s: %w", client.ShortName, err)
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "Snapshot check completed successfully")
	return nil
}

// checkAndUpdateClient checks a single client and updates snapshot if needed
func (s *SnapshotCheckerService) checkAndUpdateClient(ctx context.Context, client domain.ExecutionClientInfo) error {
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Checking client (%s)", client.ShortName, client.DnpName)

	// Wait for any ongoing tests to complete before proceeding with this client
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Checking if test is in progress...", client.ShortName)
	if err := s.waitForTestCompleteWithRetry(ctx, client.ShortName); err != nil {
		return fmt.Errorf("error while waiting for test to complete: %w", err)
	}
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] No ongoing tests detected", client.ShortName)

	// Check if a download is already in progress for this client
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Checking if download is already in progress...", client.ShortName)
	inProgress, err := s.downloadProgress.IsDownloadInProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to check download progress: %w", err)
	}
	if inProgress {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Download already in progress, skipping", client.ShortName)
		return nil
	}

	// Get latest available block number from ethpandaops
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Fetching latest available block number...", client.ShortName)
	latestBlockNumber, err := s.snapshotManager.GetLatestBlockNumber(ctx, s.config.Network, client.ShortName)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Latest available snapshot block: %s", client.ShortName, latestBlockNumber)

	// Check if we need to download by reading current block number
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Checking current snapshot block number...", client.ShortName)
	needsDownload, err := s.checkNeedsSnapshotDownload(ctx, client, latestBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to check if snapshot needed: %w", err)
	}

	if !needsDownload {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Snapshot is up to date, skipping", client.ShortName)
		return nil
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Snapshot download needed", client.ShortName)

	// Set download in progress for this client
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Setting download in progress marker...", client.ShortName)
	if err := s.downloadProgress.SetDownloadInProgress(ctx); err != nil {
		return fmt.Errorf("failed to set download in progress: %w", err)
	}

	// Ensure we clear the progress file on completion (success or failure)
	defer func() {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Clearing download in progress marker...", client.ShortName)
		if err := s.downloadProgress.ClearDownloadInProgress(ctx); err != nil {
			logger.ErrorWithPrefix(snapshotLogPrefix, "[%s] Failed to clear download in progress: %v", client.ShortName, err)
		}
	}()

	// Download and mount snapshot
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Starting snapshot download and extraction...", client.ShortName)
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Target path: %s", client.ShortName, client.VolumeTargetPath)

	start := time.Now()
	err = s.snapshotManager.DownloadAndMountSnapshot(ctx, s.config.Network, client)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to download and mount snapshot: %w", err)
	}

	// Write block number file after successful download
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Writing snapshot block number: %s", client.ShortName, latestBlockNumber)
	if err := s.blockNumber.WriteBlockNumber(ctx, latestBlockNumber); err != nil {
		return fmt.Errorf("failed to write block number: %w", err)
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] ✓ Snapshot download completed successfully", client.ShortName)
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Total time: %s", client.ShortName, elapsed.Round(time.Second))

	return nil
}

// checkNeedsSnapshotDownload determines if a snapshot needs to be downloaded
// by reading the current block number and comparing with the latest available
// Logs clearly the current and latest block numbers for visibility
func (s *SnapshotCheckerService) checkNeedsSnapshotDownload(ctx context.Context, client domain.ExecutionClientInfo, latestBlockNumber string) (bool, error) {
	// Check if block number file exists
	exists, err := s.blockNumber.BlockNumberExists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if block number exists: %w", err)
	}

	if !exists {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] No snapshot block number file found - download required", client.ShortName)
		return true, nil
	}

	// Read current block number
	currentBlockNumber, err := s.blockNumber.ReadBlockNumber(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to read current block number: %w", err)
	}

	// Log both block numbers clearly
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Current snapshot block: %s", client.ShortName, currentBlockNumber)
	logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Latest available block: %s", client.ShortName, latestBlockNumber)

	// Check if newer snapshot is available
	isNewer, err := s.blockNumber.IsNewerSnapshot(ctx, latestBlockNumber)
	if err != nil {
		return false, fmt.Errorf("failed to compare block numbers: %w", err)
	}

	if isNewer {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Newer snapshot available (%s > %s) - download required", client.ShortName, latestBlockNumber, currentBlockNumber)
	} else {
		logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Current snapshot is up to date (current: %s, latest: %s)", client.ShortName, currentBlockNumber, latestBlockNumber)
	}

	return isNewer, nil
}

// StopAllDownloads stops all running download containers (for graceful shutdown)
func (s *SnapshotCheckerService) StopAllDownloads(ctx context.Context) {
	logger.InfoWithPrefix(snapshotLogPrefix, "Stopping all snapshot download containers...")
	s.snapshotManager.StopAllDownloads(ctx)
	logger.InfoWithPrefix(snapshotLogPrefix, "All download containers stopped")
}

// ClearDownloadMarker clears the download in progress marker
func (s *SnapshotCheckerService) ClearDownloadMarker(ctx context.Context) {
	if err := s.downloadProgress.ClearDownloadInProgress(ctx); err != nil {
		logger.WarnWithPrefix(snapshotLogPrefix, "Failed to clear download marker: %v", err)
	}
}

// waitForTestCompleteWithRetry waits until no test is in progress
// using a fixed 30 second retry interval.
func (s *SnapshotCheckerService) waitForTestCompleteWithRetry(ctx context.Context, clientName string) error {
	const retryInterval = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			inProgress, err := s.testProgress.IsTestInProgress(ctx)
			if err != nil {
				return fmt.Errorf("error checking test progress: %w", err)
			}

			if !inProgress {
				return nil
			}

			logger.InfoWithPrefix(snapshotLogPrefix, "[%s] Test in progress, waiting %s before retry...", clientName, retryInterval)
			time.Sleep(retryInterval)
		}
	}
}
