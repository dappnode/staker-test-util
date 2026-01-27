package cleaner

import (
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/application/domain"
	"context"
	"fmt"
)

type CleanerAdapter struct {
	Dappmanager *dappmanager.DappManagerAdapter
	Execution   *execution.ExecutionAdapter
	Docker      *docker.DockerAdapter
	BlockNumber *blocknumber.BlockNumberAdapter
}

func NewCleanerAdapter(dappmanager *dappmanager.DappManagerAdapter, execution *execution.ExecutionAdapter, docker *docker.DockerAdapter, blockNumber *blocknumber.BlockNumberAdapter) *CleanerAdapter {
	return &CleanerAdapter{
		Dappmanager: dappmanager,
		Execution:   execution,
		Docker:      docker,
		BlockNumber: blockNumber,
	}
}

// CleanEnvironment stops containers and removes non-core packages
func (e *CleanerAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig) error {
	var errs []error

	// Get latest block number from execution client and update it in volume
	latestBlockNumber, err := e.Execution.GetLatestBlockNumber(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("get latest block number failed: %w", err))
	}
	err = e.BlockNumber.WriteBlockNumber(ctx, latestBlockNumber)
	if err != nil {
		errs = append(errs, fmt.Errorf("write block number failed: %w", err))
	}

	// Attempt to stop container
	err = e.Docker.StopContainer(ctx, stakerConfig.ExecutionContainerName)
	if err != nil {
		errs = append(errs, fmt.Errorf("stop container failed: %w", err))
	}

	// Attempt to remove non-core packages
	_, pkgErrs := e.Dappmanager.RemoveNonCorePackages(ctx)
	for _, pkgErr := range pkgErrs {
		errs = append(errs, fmt.Errorf("remove non-core package failed: %w", pkgErr))
	}

	// Return combined error if any step failed
	if len(errs) > 0 {
		return fmt.Errorf("CleanEnvironment encountered errors: %v", errs)
	}
	return nil
}
