package cleaner

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/application/domain"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/system/mount"
	"context"
	"fmt"
)

type CleanerAdapter struct {
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
	Docker      *docker.DockerAdapter
	Mount       *mount.MountAdapter

}

func NewCleanerAdapter(execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter, docker *docker.DockerAdapter, mount *mount.MountAdapter) *CleanerAdapter {
	return &CleanerAdapter{
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
		Docker:      docker,
		Mount:       mount,
	}
}

// CleanEnvironment stops the execution client and clears the validators from the brain.
func (e *CleanerAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, mountConfig domain.Mount) error {
	// TODO: clean should remove all non-core packages and also do not remove signer volumes so validator indexes are there. and release mount

	volumeTarget, err := e.Docker.StopAndGetVolumeTarget(ctx, mountConfig.Path)
	if err != nil {
		return fmt.Errorf("failed to stop container and get volume: %w", err)
	}

	if err := e.Mount.UnmountNFS(ctx, volumeTarget); err != nil {
		return fmt.Errorf("failed to mount NFS: %w", err)
	}
	if err := e.Docker.StartContainer(ctx, stakerConfig.ExecutionContainerName); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}
