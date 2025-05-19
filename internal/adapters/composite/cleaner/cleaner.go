package cleaner

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/system/mount"
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
	Mount       *mount.MountAdapter
}

func NewCleanerAdapter(dappmanager *dappmanager.DappManagerAdapter, execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter, docker *docker.DockerAdapter, mount *mount.MountAdapter) *CleanerAdapter {
	return &CleanerAdapter{
		Dappmanager: dappmanager,
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
		Docker:      docker,
		Mount:       mount,
	}
}

// CleanEnvironment release the mounted volume and remove non-core packages
func (e *CleanerAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, mountConfig domain.Mount) error {
	// Release the mounted volume
	volumeTarget, err := e.Docker.StopAndGetVolumeTarget(ctx, mountConfig.Path)
	if err == nil {
		if err := e.Mount.UnmountNFS(ctx, volumeTarget); err != nil {
			return fmt.Errorf("failed to mount NFS: %w", err)
		}
	}

	// Remove non-core packages. Web3signer volume is not removed to persist the keys
	err = e.Dappmanager.RemoveNonCorePackages(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove non-core packages: %w", err)
	}

	return nil
}
