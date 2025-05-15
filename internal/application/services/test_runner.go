package services

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/application/ports"

	"context"
	"fmt"
)

type TestRunnerService struct {
	Ensurer  ports.EnvironmentEnsurer
	Executor ports.TestExecutor
	Cleaner  ports.EnvironmentCleaner
}

func NewTestRunner(
	en ports.EnvironmentEnsurer,
	ex ports.TestExecutor,
	cl ports.EnvironmentCleaner,
) *TestRunnerService {
	return &TestRunnerService{en, ex, cl}
}

// RunTest wires up the three steps in sequence.
// It ensures the environment, executes the test, and cleans up.
func (s *TestRunnerService) RunTest(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// 1) ensure environment
	if err := s.Ensurer.EnsureEnvironment(ctx, mountConfig, stakerConfig, pkg); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// 2) execute test
	execErr := s.Executor.ExecuteTest(ctx)
	if execErr != nil {
		// even on test failure we want cleanup
		if cleanupErr := s.Cleaner.CleanEnvironment(mountConfig); cleanupErr != nil {
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		return fmt.Errorf("test failed: %w", execErr)
	}

	// 3) cleanup
	if err := s.Cleaner.CleanEnvironment(mountConfig); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}
