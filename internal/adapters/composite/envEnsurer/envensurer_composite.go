package envensurer

import (
	"context"
	"fmt"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/mount"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/application/domain"
)

// cli args: ipfsHash
// Fronm the IPFS hash it will be obtained the dnpName
// From the dnpName it will be obtained the network
// From the network it will be obtained the staker config randomized
// ideas: save timestart and end and collect docker logs and report them

type EnvEnsurerAdapter struct {
	DappManager  *dappmanager.DappManagerAdapter
	Brain        *brain.BrainAdapter
	Tropidatooor *tropidatooor.TropidatooorAdapter
	Docker       *docker.DockerAdapter
	Mount        *mount.MountAdapter
	Beaconchain  *beaconchain.BeaconchainAdapter
	Execution    *execution.ExecutionAdapter
	Ipfs         *ipfs.IPFSAdapter
}

func NewEnvEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, tropidatooor *tropidatooor.TropidatooorAdapter, docker *docker.DockerAdapter, mountAdapter *mount.MountAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter, ipfs *ipfs.IPFSAdapter) *EnvEnsurerAdapter {
	return &EnvEnsurerAdapter{
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
func (e *EnvEnsurerAdapter) EnsureEnvironment(ctx context.Context, ipfsHash string, config domain.TestConfig) error {
	if err := e.DappManager.SetStakerConfig(ctx, config.StakerClients); err != nil {
		return fmt.Errorf("failed to set staker config for DNP: %w", err)
	}
	if err := e.DappManager.PackageInstall(ctx, config.DnpName, ipfsHash); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}
	volumeTarget, err := e.Docker.StopAndGetVolumeTarget(ctx, config.ExecutionContainerName)
	if err != nil {
		return fmt.Errorf("failed to stop container and get volume: %w", err)
	}

	if err := e.Mount.MountNFS(ctx, config.MountPath, volumeTarget); err != nil {
		return fmt.Errorf("failed to mount NFS: %w", err)
	}
	if err := e.Docker.StartContainer(ctx, config.ExecutionContainerName); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}
