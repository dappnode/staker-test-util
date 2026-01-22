package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"time"
)

var logPrefix = "TestRunner"

type TestRunnerService struct {
	Runner   ports.TestRunner
	Download ports.DownloadProgress
}

func NewTestRunner(runner ports.TestRunner, download ports.DownloadProgress) *TestRunnerService {
	return &TestRunnerService{Runner: runner, Download: download}
}

// RunTest wires up the three steps in sequence.
// It ensures the environment, executes the test, and cleans up.
func (s *TestRunnerService) RunTest(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// 1) wait for any ongoing downloads to complete
	logger.InfoWithPrefix(logPrefix, "Waiting for any ongoing downloads to complete...")
	if err := s.WaitForDownloadCompleteWithRetry(ctx); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Error while waiting for downloads to complete: %v", err)
		return fmt.Errorf("error while waiting for downloads to complete: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "No ongoing downloads detected, proceeding with test run")

	// 2) ensure environment
	logger.InfoWithPrefix(logPrefix, "Step 2: Ensuring environment for package %s", pkg.DnpName)
	if err := s.Runner.EnsureEnvironment(ctx, stakerConfig, pkg); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Setup failed: %v", err)
		return fmt.Errorf("setup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Environment setup completed successfully")

	// 3) execute test (now includes report generation and PR commenting)
	logger.InfoWithPrefix(logPrefix, "Step 3: Executing test")
	execErr := s.Runner.ExecuteTest(ctx, stakerConfig)
	if execErr != nil {
		logger.ErrorWithPrefix(logPrefix, "Test execution failed: %v", execErr)
		// even on test failure we want cleanup
		logger.InfoWithPrefix(logPrefix, "Step 4: Running cleanup after test failure")
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup also failed: %v", cleanupErr)
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		return fmt.Errorf("test failed: %w", execErr)
	}
	logger.InfoWithPrefix(logPrefix, "Test execution completed successfully")

	// 4) cleanup
	logger.InfoWithPrefix(logPrefix, "Step 4: Cleaning up environment")
	if err := s.Runner.CleanEnvironment(ctx, stakerConfig); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", err)
		return fmt.Errorf("cleanup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
	return nil
}

// WaitForDownloadCompleteWithRetry waits until no download is in progress
// using a fixed 30 second retry interval. This is useful for the test runner
// to wait for snapshot downloads to complete before proceeding.
func (s *TestRunnerService) WaitForDownloadCompleteWithRetry(ctx context.Context) error {
	const retryInterval = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			inProgress, err := s.Download.IsDownloadInProgress(ctx)
			if err != nil {
				return fmt.Errorf("error checking download progress: %w", err)
			}

			if !inProgress {
				return nil
			}

			time.Sleep(retryInterval)
		}
	}
}
