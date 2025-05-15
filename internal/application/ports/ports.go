package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

type EnvironmentEnsurer interface {
	EnsureEnvironment(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error
}

type TestExecutor interface {
	ExecuteTest(ctx context.Context) error
}

type EnvironmentCleaner interface {
	CleanEnvironment(mountConfig domain.Mount) error
}
