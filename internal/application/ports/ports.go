package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

type EnvironmentGetter interface {
	GetEnvironmentConfig(ctx context.Context, ipfsHash string) (*domain.TestConfig, error)
}

type EnvironmentEnsurer interface {
	EnsureEnvironment(ctx context.Context, ipfsHash string, config domain.TestConfig) error
}

type TestExecutor interface {
	ExecuteTest(ctx context.Context, indexes []string) error
}

type EnvironmentCleaner interface {
	CleanEnvironment(config *domain.TestConfig) error
}
