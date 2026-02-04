package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

type TestRunner interface {
	EnsureEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg) error
	ExecuteTest(ctx context.Context, stakerConfig domain.StakerConfig) error
	ExecuteSync(ctx context.Context, stakerConfig domain.StakerConfig) error
	CleanEnvironment(context.Context, domain.StakerConfig) error
}
