package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"sync"
	"time"
)

var snapshotLogPrefix = "SnapshotChecker"

// SnapshotCheckerService handles snapshot checking and downloading
type SnapshotCheckerService struct {
	snapshotManager ports.SnapshotManager
	progress        ports.DownloadProgress
	config          domain.SnapshotCheckerConfig
}

// NewSnapshotCheckerService creates a new SnapshotCheckerService
func NewSnapshotCheckerService(
	snapshotManager ports.SnapshotManager,
	progress ports.DownloadProgress,
	config domain.SnapshotCheckerConfig,
) *SnapshotCheckerService {
	return &SnapshotCheckerService{
		snapshotManager: snapshotManager,
		progress:        progress,
		config:          config,
	}
}

// GetSnapshotManager returns the snapshot manager adapter (for shutdown handling)
func (s *SnapshotCheckerService) GetSnapshotManager() ports.SnapshotManager {
	return s.snapshotManager
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
	latestBlockNumber, err := s.snapshotManager.GetLatestBlockNumber(ctx, s.config.Network, client.ShortName)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	logger.InfoWithPrefix(snapshotLogPrefix, "Latest available snapshot for %s: block %s", client.ShortName, latestBlockNumber)

	// Check if we need to download (delegated to composite adapter)
	needsDownload, err := s.snapshotManager.NeedsSnapshotDownload(ctx, client, latestBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to check if snapshot needed: %w", err)
	}

	if !needsDownload {
		logger.InfoWithPrefix(snapshotLogPrefix, "Snapshot for %s is up to date, skipping", client.ShortName)
		return nil
	}

	// Download and mount snapshot (delegated to composite adapter)
	if err := s.snapshotManager.DownloadAndMountSnapshot(ctx, s.config.Network, client); err != nil {
		return fmt.Errorf("failed to download and mount snapshot: %w", err)
	}

	return nil
}
