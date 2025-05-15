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
// It retrieves the config, ensures the environment, executes the test, and cleans up.
func (s *TestRunnerService) RunTest(ctx context.Context, ipfsHash string) error {
	// 1) get config
	getter, ok := s.Ensurer.(ports.EnvironmentGetter)
	if !ok {
		return fmt.Errorf("ensurer does not implement EnvironmentGetter")
	}
	config, err := getter.GetEnvironmentConfig(ctx, ipfsHash)
	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	// 2) ensure environment
	if err := s.Ensurer.EnsureEnvironment(ctx, ipfsHash, *config); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// 3) execute test
	execErr := s.Executor.ExecuteTest(ctx, config.Staker.ValidatorIndexes)
	if execErr != nil {
		// even on test failure we want cleanup
		if cleanupErr := s.Cleaner.CleanEnvironment(config); cleanupErr != nil {
			return fmt.Errorf("test failed: %v; cleanup also failed: %w", execErr, cleanupErr)
		}
		return fmt.Errorf("test failed: %w", execErr)
	}

	// 4) cleanup
	if err := s.Cleaner.CleanEnvironment(config); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}
