package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

type TestRunner interface {
	EnsureEnvironment(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error
	ExecuteTest(ctx context.Context) error
	CleanEnvironment(context.Context, domain.StakerConfig, domain.Mount) error
}
