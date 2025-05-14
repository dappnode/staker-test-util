package services

import (
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
func (s *TestRunnerService) RunTest(ctx context.Context) error {
	// 1) ensure
	if err := s.Ensurer.EnsureEnvironment(ctx); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// 2) execute
	execErr := s.Executor.ExecuteTest(ctx)
	if execErr != nil {
		// even on test failure we want cleanup
		if cleanupErr := s.Cleaner.CleanUpEnvironment(ctx); cleanupErr != nil {
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		return fmt.Errorf("test failed: %w", execErr)
	}

	// 3) cleanup
	if err := s.Cleaner.CleanUpEnvironment(ctx); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}
