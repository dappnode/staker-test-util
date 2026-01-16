package cleaner

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/application/domain"
	"context"
	"fmt"
)

type CleanerAdapter struct {
	Dappmanager *dappmanager.DappManagerAdapter
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
	Docker      *docker.DockerAdapter
}

func NewCleanerAdapter(dappmanager *dappmanager.DappManagerAdapter, execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter, docker *docker.DockerAdapter) *CleanerAdapter {
	return &CleanerAdapter{
		Dappmanager: dappmanager,
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
		Docker:      docker,
	}
}

// CleanEnvironment stops containers and removes non-core packages
func (e *CleanerAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig) error {
	var errs []error

	// Attempt to stop container
	_, err := e.Docker.StopAndGetVolumeTarget(ctx, stakerConfig.ExecutionContainerName, stakerConfig.ExecutionVolumeName)
	if err != nil {
		errs = append(errs, fmt.Errorf("stop container failed: %w", err))
	}

	// Attempt to remove non-core packages
	pkgErrs := e.Dappmanager.RemoveNonCorePackages(ctx)
	for _, pkgErr := range pkgErrs {
		errs = append(errs, fmt.Errorf("remove non-core package failed: %w", pkgErr))
	}

	// Return combined error if any step failed
	if len(errs) > 0 {
		return fmt.Errorf("CleanEnvironment encountered errors: %v", errs)
	}
	return nil
}
