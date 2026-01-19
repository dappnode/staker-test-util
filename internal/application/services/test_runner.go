package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"
	"context"
	"fmt"
)

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
	if err := s.Runner.EnsureEnvironment(ctx, stakerConfig, pkg); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// 2) execute test (now includes report generation and PR commenting)
	execErr := s.Runner.ExecuteTest(ctx, stakerConfig)
	if execErr != nil {
		// even on test failure we want cleanup
		if cleanupErr := s.Runner.CleanEnvironment(ctx, stakerConfig); cleanupErr != nil {
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		return fmt.Errorf("test failed: %w", execErr)
	}

	// 3) cleanup
	if err := s.Runner.CleanEnvironment(ctx, stakerConfig); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}

// GetReport returns the test report from the runner
func (s *TestRunnerService) GetReport() *domain.TestReport {
	return s.Runner.GetReport()
}
