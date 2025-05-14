package config

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/application/domain"
)

type ConfigCompositeAdapter struct {
	Brain        *brain.BrainAdapter
	Beaconchain  *beaconchain.BeaconchainAdapter
	Ipfs         *ipfs.IPFSAdapter
	Tropidatooor *tropidatooor.TropidatooorAdapter
}

func NewConfigCompositeAdapter(brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter, ipfs *ipfs.IPFSAdapter, tropidatooor *tropidatooor.TropidatooorAdapter) *ConfigCompositeAdapter {
	return &ConfigCompositeAdapter{
		Brain:        brain,
		Beaconchain:  beaconchain,
		Ipfs:         ipfs,
		Tropidatooor: tropidatooor,
	}
}

func (c *ConfigCompositeAdapter) GetTestConfig(ctx context.Context, ipfsHash string) (*domain.TestConfig, error) {
	pubkeys, err := c.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch validators from brain: %v", err)
	}
	if len(pubkeys) == 0 {
		return nil, fmt.Errorf("at least 1 validator must be loaded to be able to run the test")
	}
	indexes, err := c.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators indexes: %w", err)
	}
	dnpName, err := c.Ipfs.GetDnpNameFromHash(ctx, ipfsHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get dnpName from IPFS hash: %w", err)
	}
	mountPath, mountId, err := c.Tropidatooor.GetMountPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount path: %w", err)
	}
	stakerClients := getStakerClientsForDnp(dnpName)

	// Get execution container name
	serviceName, err := c.Ipfs.GetComposeServiceName(ctx, ipfsHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get compose service name: %w", err)
	}
	execContainerName := ExecutionContainerName(serviceName, stakerClients.ExecutionDnpName)

	return &domain.TestConfig{
		ValidatorIndexes:       indexes,
		DnpName:                dnpName,
		MountPath:              mountPath,
		MountId:                mountId,
		ExecutionContainerName: execContainerName,
		StakerClients:          stakerClients,
	}, nil
}

func getStakerClientsForDnp(dnpName string) domain.StakerClients {
	network := getNetworkFromDnpName(dnpName)
	var execClients, consClients []string
	var web3signer, mevboost string
	var relays []string = nil

	switch network {
	case "gnosis":
		execClients = []string{"nethermind-xdai.dnp.dappnode.eth", "gnosis-erigon.dnp.dappnode.eth"}
		consClients = []string{"lighthouse-gnosis.dnp.dappnode.eth", "teku-gnosis.dnp.dappnode.eth", "nimbus-gnosis.dnp.dappnode.eth", "lodestar-gnosis.dnp.dappnode.eth"}
		web3signer = "web3signer-hoodi.dnp.dappnode.eth"
		mevboost = "mev-boost-hoodi.dnp.dappnode.eth"
		relays = []string{}
	case "mainnet":
		execClients = []string{"nethermind.public.dappnode.eth", "geth.dnp.dappnode.eth", "erigon.dnp.dappnode.eth", "reth.dnp.dappnode.eth", "besu.public.dappnode.eth"}
		consClients = []string{"lighthouse.dnp.dappnode.eth", "prysm.dnp.dappnode.eth", "lodestar.dnp.dappnode.eth", "nimbus.dnp.dappnode.eth", "teku.dnp.dappnode.eth"}
		web3signer = "web3signer.dnp.dappnode.eth"
		mevboost = "mev-boost.dnp.dappnode.eth"
		relays = []string{}
	case "lukso":
		execClients = []string{"lukso-geth.dnp.dappnode.eth"}
		consClients = []string{"prysm-lukso.dnp.dappnode.eth", "teku-luks.dnp.dappnode.eth"}
		web3signer = "web3signer-lukso.dnp.dappnode.eth"
		mevboost = "mev-boost-lukso.dnp.dappnode.eth"
		relays = []string{}
	case "hoodi":
		execClients = []string{"hoodi-reth.dnp.dappnode.eth", "hoodi-geth.dnp.dappnode.eth", "hoodi-besu.dnp.dappnode.eth", "hoodi-erigon.dnp.dappnode.eth", "hoodi-nethermind.dnp.dappnode.eth"}
		consClients = []string{"prysm-hoodi.dnp.dappnode.eth", "teku-hoodi.dnp.dappnode.eth", "nimbus-hoodi.dnp.dappnode.eth", "lodestar-hoodi.dnp.dappnode.eth", "lighthouse-hoodi.dnp.dappnode.eth"}
		web3signer = "web3signer-hoodi.dnp.dappnode.eth"
		mevboost = "mev-boost-hoodi.dnp.dappnode.eth"
		relays = []string{}
	}

	exec := matchOrRandom(dnpName, execClients)
	cons := matchOrRandom(dnpName, consClients)
	return domain.StakerClients{
		ExecutionDnpName:  exec,
		ConsensusDnpName:  cons,
		Web3SignerDnpName: web3signer,
		MevBoostDnpName:   mevboost,
		Relays:            relays,
		Network:           network,
	}
}

func getNetworkFromDnpName(dnpName string) string {
	name := strings.ToLower(dnpName)
	switch {
	case strings.Contains(name, "gnosis"):
		return "gnosis"
	case strings.Contains(name, "hoodi"):
		return "hoodi"
	case strings.Contains(name, "lukso"):
		return "lukso"
	default:
		return "mainnet"
	}
}

func matchOrRandom(dnpName string, candidates []string) string {
	for _, c := range candidates {
		if strings.Contains(dnpName, c) {
			return c
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return candidates[r.Intn(len(candidates))]
}

// Utility to get the short dnp name (strip .dnp.dappnode.eth)
func shortDnpName(dnpName string) string {
	return strings.TrimSuffix(dnpName, ".dnp.dappnode.eth")
}

// Utility to get the execution container name from service and dnpName
func ExecutionContainerName(serviceName, dnpName string) string {
	return fmt.Sprintf("DAppNodePackage-%s.%s.dnp.dappnode.eth", serviceName, shortDnpName(dnpName))
}
