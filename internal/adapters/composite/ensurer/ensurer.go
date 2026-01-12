package ensurer

import (
	"context"
	"fmt"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/adapters/system/mount"
	"clients-test/internal/application/domain"
)

type EnsurerAdapter struct {
	DappManager  *dappmanager.DappManagerAdapter
	Brain        *brain.BrainAdapter
	Tropidatooor *tropidatooor.TropidatooorAdapter
	Docker       *docker.DockerAdapter
	Mount        *mount.MountAdapter
	Beaconchain  *beaconchain.BeaconchainAdapter
	Execution    *execution.ExecutionAdapter
	Ipfs         *ipfs.IPFSAdapter
}

func NewEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, tropidatooor *tropidatooor.TropidatooorAdapter, docker *docker.DockerAdapter, mountAdapter *mount.MountAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter, ipfs *ipfs.IPFSAdapter) *EnsurerAdapter {
	return &EnsurerAdapter{
		DappManager:  dappManager,
		Brain:        brain,
		Tropidatooor: tropidatooor,
		Docker:       docker,
		Mount:        mountAdapter,
		Beaconchain:  beaconchain,
		Execution:    execution,
		Ipfs:         ipfs,
	}
}

// EnsureEnvironment validates the environment and prepares it for testing.
// It sets the staker config, installs the package, stops the container, mounts the NFS, and starts the container.
func (e *EnsurerAdapter) EnsureEnvironment(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	if err := e.DappManager.SetStakerConfig(ctx, stakerConfig); err != nil {
		return fmt.Errorf("failed to set staker config for DNP: %w", err)
	}
	if err := e.DappManager.PackageInstall(ctx, pkg); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}
	volumeTarget, err := e.Docker.StopAndGetVolumeTarget(ctx, stakerConfig.ExecutionContainerName, stakerConfig.ExecutionVolumeName)
	if err != nil {
		return fmt.Errorf("failed to stop container and get volume: %w", err)
	}

	if err := e.Mount.MountNFS(ctx, mountConfig.Path, volumeTarget); err != nil {
		return fmt.Errorf("failed to mount NFS: %w", err)
	}
	if err := e.Docker.StartContainer(ctx, stakerConfig.ExecutionContainerName); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}
