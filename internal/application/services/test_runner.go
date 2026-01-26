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
	Runner           ports.TestRunner
	DownloadProgress ports.DownloadProgress
	TestProgress     ports.TestProgress
}

func NewTestRunner(runner ports.TestRunner, downloadProgress ports.DownloadProgress, testProgress ports.TestProgress) *TestRunnerService {
	return &TestRunnerService{Runner: runner, DownloadProgress: downloadProgress, TestProgress: testProgress}
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

	// Set test in progress marker
	logger.InfoWithPrefix(logPrefix, "Setting test in progress marker...")
	if err := s.TestProgress.SetTestInProgress(ctx); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Failed to set test in progress: %v", err)
		return fmt.Errorf("failed to set test in progress: %w", err)
	}

	// Ensure we clean up and clear the test in progress marker on completion (success or failure)
	defer func() {
		logger.InfoWithPrefix(logPrefix, "Cleaning up environment...")
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", cleanupErr)
		} else {
			logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		}

		logger.InfoWithPrefix(logPrefix, "Clearing test in progress marker...")
		if err := s.TestProgress.ClearTestInProgress(ctx); err != nil {
			logger.ErrorWithPrefix(logPrefix, "Failed to clear test in progress: %v", err)
		}
	}()

	// 2) ensure environment
	logger.InfoWithPrefix(logPrefix, "Step 2: Ensuring environment for package %s", pkg.DnpName)
	if err := s.Runner.EnsureEnvironment(ctx, stakerConfig, pkg); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Setup failed: %v", err)
		return fmt.Errorf("setup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Environment setup completed successfully")

	// 3) execute test (now includes report generation and PR commenting)
	logger.InfoWithPrefix(logPrefix, "Step 3: Executing test")
	if err := s.Runner.ExecuteTest(ctx, stakerConfig); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Test execution failed: %v", err)
		return fmt.Errorf("test failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Test execution completed successfully")

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
			inProgress, err := s.DownloadProgress.IsDownloadInProgress(ctx)
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
