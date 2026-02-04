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

// RunTest wires up the three steps in sequence.
// It ensures the environment, executes the test, and cleans up.
func (s *TestRunnerService) RunTest(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// Ensure we clean up and clear the test in progress marker on completion (success or failure)
	defer func() {
		logger.InfoWithPrefix(logPrefix, "Cleaning up environment...")
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", cleanupErr)
		} else {
			logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
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
