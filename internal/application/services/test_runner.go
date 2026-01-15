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
func (s *TestRunnerService) RunTest(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// 1) ensure environment
	if err := s.Runner.EnsureEnvironment(ctx, mountConfig, stakerConfig, pkg); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// 2) execute test
	execErr := s.Runner.ExecuteTest(ctx)
	if execErr != nil {
		return fmt.Errorf("test failed: %w", execErr)
	}

	return nil
}
