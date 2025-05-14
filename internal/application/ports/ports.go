package ports

import "context"

// Prepare any needed VMs, containers, secrets, etc.
type EnvironmentEnsurer interface {
	EnsureEnvironment(ctx context.Context) error
}

// Execute the actual test suite, return pass/fail
type TestExecutor interface {
	ExecuteTest(ctx context.Context) error
}

// Tear down whatever you stood up (always run, even on failure)
type EnvironmentCleaner interface {
	CleanUpEnvironment(ctx context.Context) error
}
