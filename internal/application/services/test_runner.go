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
	return &TestRunnerService{runner}
}

// RunTest wires up the three steps in sequence.
// It ensures the environment, executes the test, and cleans up.
func (s *TestRunnerService) RunTest(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// 1) ensure environment
	logger.InfoWithPrefix(logPrefix, "Step 1: Ensuring environment for package %s", pkg.DnpName)
	if err := s.Runner.EnsureEnvironment(ctx, stakerConfig, pkg); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Setup failed: %v", err)
		return fmt.Errorf("setup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Environment setup completed successfully")

	// 2) execute test (now includes report generation and PR commenting)
	logger.InfoWithPrefix(logPrefix, "Step 2: Executing test")
	execErr := s.Runner.ExecuteTest(ctx, stakerConfig)
	if execErr != nil {
		logger.ErrorWithPrefix(logPrefix, "Test execution failed: %v", execErr)
		// even on test failure we want cleanup
		logger.InfoWithPrefix(logPrefix, "Step 3: Running cleanup after test failure")
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup also failed: %v", cleanupErr)
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		return fmt.Errorf("test failed: %w", execErr)
	}
	logger.InfoWithPrefix(logPrefix, "Test execution completed successfully")

	// 3) cleanup
	logger.InfoWithPrefix(logPrefix, "Step 3: Cleaning up environment")
	if err := s.Runner.CleanEnvironment(ctx, stakerConfig); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", err)
		return fmt.Errorf("cleanup failed: %w", err)
	}
	logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
	return nil
}
