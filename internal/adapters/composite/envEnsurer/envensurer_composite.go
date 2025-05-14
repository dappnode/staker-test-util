package envensurer

import (
	"context"
	"fmt"
	"strings"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/mount"
	"clients-test/internal/adapters/apis/tropidatooor"
)

type EnvEnsurerAdapter struct {
	DappManager  *dappmanager.DappManagerAdapter
	Brain        *brain.BrainAdapter
	Tropidatooor *tropidatooor.TropidatooorAdapter
	Docker       *docker.DockerAdapter
	Mount        *mount.MountAdapter
	Beaconchain  *beaconchain.BeaconchainAdapter
	Execution    *execution.ExecutionAdapter
}

func NewEnvEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, tropidatooor *tropidatooor.TropidatooorAdapter, docker *docker.DockerAdapter, mountAdapter *mount.MountAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter) *EnvEnsurerAdapter {
	return &EnvEnsurerAdapter{
		DappManager:  dappManager,
		Brain:        brain,
		Tropidatooor: tropidatooor,
		Docker:       docker,
		Mount:        mountAdapter,
		Beaconchain:  beaconchain,
		Execution:    execution,
	}
}

// EnsureEnvironment validates the environment and prepares it for testing.
// Steps:
// 1. Ensures at least one validator is loaded in brain.
// 2. Sets the staker config in dappmanager based on dnpName type.
// 3. Installs the package via dappmanager.
// 4. Retrieves the mount path from tropidatooor.
// 5. Remounts the docker volume for the execution client.
func (e *EnvEnsurerAdapter) EnsureEnvironment(ctx context.Context, ipfsHash, dnpName string) (mountId string, indexes []string, err error) {
	pubkeys, err := e.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch validators from brain: %v", err)
	}
	if len(pubkeys) == 0 {
		return "", nil, fmt.Errorf("at least 1 validator must be loaded to be able to run the test")
	}
	indexes, err = e.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get validators indexes: %w", err)
	}
	if err := e.setStakerConfigForDnp(ctx, dnpName); err != nil {
		return "", nil, fmt.Errorf("failed to set staker config for DNP: %w", err)
	}
	if err := e.DappManager.PackageInstall(ctx, dnpName, ipfsHash); err != nil {
		return "", nil, fmt.Errorf("failed to install package: %w", err)
	}
	mountPath, mountId, err := e.Tropidatooor.GetMountPath(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get mount path: %w", err)
	}
	if err := e.remountExecutionVolume(ctx, mountPath); err != nil {
		return "", nil, fmt.Errorf("failed to remount execution volume: %w", err)
	}
	return mountId, indexes, nil
}

func (e *EnvEnsurerAdapter) setStakerConfigForDnp(ctx context.Context, dnpName string) error {
	network := dappmanager.Hoodi
	execName := "hoodi-geth.dnp.dappnode.eth"
	consName := "prysm-hoodi.dnp.dappnode.eth"
	signerName := "web3signer-hoodi.dnp.dappnode.eth"
	mevBoostName := "mev-boost-hoodi.dnp.dappnode.eth"

	switch {
	case containsAny(dnpName, []string{"geth", "besu", "erigon", "reth", "nethermind"}):
		execName = dnpName
	case containsAny(dnpName, []string{"prysm", "nimbus", "teku", "lighthouse"}):
		consName = dnpName
	case strings.Contains(dnpName, "web3signer"):
		signerName = dnpName
	case strings.Contains(dnpName, "mev-boost"):
		mevBoostName = dnpName
	}

	return e.DappManager.SetStakerConfig(
		ctx,
		network,
		&execName,
		&consName,
		&mevBoostName,
		&signerName,
		[]string{},
	)
}

func (e *EnvEnsurerAdapter) remountExecutionVolume(ctx context.Context, mountPath string) error {
	containerName := "DAppNodePackage-hoodi-geth.dnp.dappnode.eth"
	volumeTarget, err := e.Docker.StopAndGetVolumeTarget(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to stop container and get volume: %w", err)
	}
	if err := e.Mount.MountNFS(ctx, mountPath, volumeTarget); err != nil {
		return fmt.Errorf("failed to mount NFS: %w", err)
	}
	if err := e.Docker.StartContainer(ctx, containerName); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
