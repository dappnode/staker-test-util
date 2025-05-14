package envensurer

import (
	"context"
	"fmt"

	brain "clients-test/internal/adapters/apis/brain"
	dappmanager "clients-test/internal/adapters/apis/dappmanager"
)

type EnvEnsurerAdapter struct {
	DappManager *dappmanager.DappManagerAdapter
	Brain       *brain.BrainAdapter
}

func NewEnvEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter) *EnvEnsurerAdapter {
	return &EnvEnsurerAdapter{
		DappManager: dappManager,
		Brain:       brain,
	}
}

// EnsureEnvironment checks that dappmanager is available and at least one validator is loaded in brain, with context support
func (e *EnvEnsurerAdapter) EnsureEnvironment(ctx context.Context) error {
	if err := e.DappManager.Ping(ctx); err != nil {
		return fmt.Errorf("it seems like the test api of the dappmanager is not available, make sure dappmanager is running with TEST env set to true: %v", err)
	}
	pubkeys, err := e.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch validators from brain: %v", err)
	}
	if len(pubkeys) == 0 {
		return fmt.Errorf("at least 1 validator must be loaded to be able to run the test")
	}
	return nil
}
