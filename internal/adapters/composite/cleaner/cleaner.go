package cleaner

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/application/domain"
)

type CleanerAdapter struct {
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
}

func NewCleanerAdapter(execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter) *CleanerAdapter {
	return &CleanerAdapter{
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
	}
}

// CleanEnvironment stops the execution client and clears the validators from the brain.
func (e *CleanerAdapter) CleanEnvironment(mountConfig domain.Mount) error {
	return nil
}
