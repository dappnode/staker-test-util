package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"
	"clients-test/internal/logger"
	"context"
	"fmt"
)

var logPrefix = "TestRunner"

type TestRunnerService struct {
	Runner ports.TestRunner
}

func NewTestRunner(runner ports.TestRunner) *TestRunnerService {
	return &TestRunnerService{Runner: runner}
}

// Run executes the workflow based on the specified mode.
// For sync mode: ensures environment, waits for sync, then cleans up.
// For test mode: ensures environment, runs full attestation test, then cleans up.
func (s *TestRunnerService) Run(ctx context.Context, mode domain.RunMode, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// Ensure we clean up on completion (success or failure)
	defer func() {
		logger.InfoWithPrefix(logPrefix, "Cleaning up environment...")
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", cleanupErr)
		} else {
			logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		}
	}()

	// Step 1: Ensure environment
	logger.InfoWithPrefix(logPrefix, "Step 1: Ensuring environment for package %s", pkg.DnpName)
	if err := s.Runner.EnsureEnvironment(ctx, stakerConfig, pkg); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Setup failed: %v", err)
		return fmt.Errorf("setup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Environment setup completed successfully")

	// Step 2: Execute based on mode
	switch mode {
	case domain.ModeSync:
		logger.InfoWithPrefix(logPrefix, "Step 2: Executing sync mode (waiting for clients to sync)")
		if err := s.Runner.ExecuteSync(ctx, stakerConfig); err != nil {
			logger.ErrorWithPrefix(logPrefix, "Sync failed: %v", err)
			return fmt.Errorf("sync failed: %w", err)
		}
		logger.InfoWithPrefix(logPrefix, "Sync completed successfully")

	case domain.ModeTest:
		logger.InfoWithPrefix(logPrefix, "Step 2: Executing test mode (sync + attestation test)")
		if err := s.Runner.ExecuteTest(ctx, stakerConfig); err != nil {
			logger.ErrorWithPrefix(logPrefix, "Test execution failed: %v", err)
			return fmt.Errorf("test failed: %w", err)
		}
		logger.InfoWithPrefix(logPrefix, "Test execution completed successfully")

	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	logger.InfoWithPrefix(logPrefix, "Run completed successfully in %s mode", mode)
	return nil
}
