package envcleaner

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/application/domain"
)

type EnvCleanerAdapter struct {
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
}

func NewEnvCleanerAdapter(execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter) *EnvCleanerAdapter {
	return &EnvCleanerAdapter{
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
	}
}

// CleanEnvironment stops the execution client and clears the validators from the brain.
func (e *EnvCleanerAdapter) CleanEnvironment(mountConfig domain.Mount) error {
	return nil
}
