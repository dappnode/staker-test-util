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
	"clients-test/internal/adapters/apis/tropidatooor"
)

type EnvEnsurerAdapter struct {
	DappManager  *dappmanager.DappManagerAdapter
	Brain        *brain.BrainAdapter
	Tropidatooor *tropidatooor.TropidatooorAdapter
	Docker       *docker.DockerAdapter
	Beaconchain  *beaconchain.BeaconchainAdapter
	Execution    *execution.ExecutionAdapter
}

func NewEnvEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, tropidatooor *tropidatooor.TropidatooorAdapter, docker *docker.DockerAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter) *EnvEnsurerAdapter {
	return &EnvEnsurerAdapter{
		DappManager:  dappManager,
		Brain:        brain,
		Tropidatooor: tropidatooor,
		Docker:       docker,
		Beaconchain:  beaconchain,
		Execution:    execution,
	}
}

// EnsureEnvironment checks that dappmanager is available and at least one validator is loaded in brain, with context support
func (e *EnvEnsurerAdapter) EnsureEnvironment(ctx context.Context, ipfsHash, dnpName string) error {
	// Sanity check: at least one validator must be loaded in brain
	pubkeys, err := e.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch validators from brain: %v", err)
	}
	if len(pubkeys) == 0 {
		return fmt.Errorf("at least 1 validator must be loaded to be able to run the test")
	}

	// 1. Set staker config
	network := dappmanager.Hoodi
	execution := "hoodi-geth.dnp.dappnode.eth"
	consensus := "prysm-hoodi.dnp.dappnode.eth"
	web3signer := "web3signer-hoodi.dnp.dappnode.eth"
	mevboost := "mev-boost-hoodi.dnp.dappnode.eth"

	// Determine which field to set based on dnpName
	var execPtr, consPtr, mevPtr, signerPtr *string
	switch {
	case containsAny(dnpName, []string{"geth", "besu", "erigon", "reth", "nethermind"}):
		execution = dnpName
		execPtr = &execution
		consPtr = &consensus
		mevPtr = &mevboost
		signerPtr = &web3signer
	case containsAny(dnpName, []string{"prysm", "nimbus", "teku", "lighthouse"}):
		consensus = dnpName
		execPtr = &execution
		consPtr = &consensus
		mevPtr = &mevboost
		signerPtr = &web3signer
	case containsAny(dnpName, []string{"web3signer"}):
		web3signer = dnpName
		execPtr = &execution
		consPtr = &consensus
		mevPtr = &mevboost
		signerPtr = &web3signer
	case containsAny(dnpName, []string{"mev-boost"}):
		mevboost = dnpName
		execPtr = &execution
		consPtr = &consensus
		mevPtr = &mevboost
		signerPtr = &web3signer
	default:
		execPtr = &execution
		consPtr = &consensus
		mevPtr = &mevboost
		signerPtr = &web3signer
	}

	if err := e.DappManager.SetStakerConfig(ctx, network, execPtr, consPtr, mevPtr, signerPtr, []string{}); err != nil {
		return fmt.Errorf("failed to set staker config: %w", err)
	}

	// 2. Install package
	if err := e.DappManager.PackageInstall(ctx, dnpName, ipfsHash); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// 3. Get mount path
	mountPath, err := e.Tropidatooor.GetMountPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get mount path: %w", err)
	}

	// 4. Remount docker volume
	containerName := "DAppNodePackage-hoodi-geth.dnp.dappnode.eth"
	if err := e.Docker.RemountDockerVolume(ctx, containerName, mountPath); err != nil {
		return fmt.Errorf("failed to remount docker volume: %w", err)
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
