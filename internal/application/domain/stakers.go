package domain

import (
	"math/rand"
	"strings"
	"time"
)

type StakerConfig struct {
	ExecutionDnpName         string   `json:"executionDnpName"`
	ConsensusDnpName         string   `json:"consensusDnpName"`
	Web3SignerDnpName        string   `json:"web3signerDnpName"`
	MevBoostDnpName          string   `json:"mevBoostDnpName"`
	Relays                   []string `json:"relays,omitempty"` // Optional, can be empty
	Network                  string   `json:"network"`          // The network this config is for (e.g., mainnet, gnosis, hoodi, lukso)
	Urls                     Urls
	BrainContainerName       string // The name of the brain container
	SignerContainerName      string // The name of the web3signer container
	BeaconchainContainerName string // The name of the beaconchain container
	ValidatorContainerName   string // The name of the validator container
	ExecutionContainerName   string // The name of the container to mount the NFS volume to
}

type Urls struct {
	ExecutionURL   string
	BrainURL       string
	BeaconchainURL string
	DappmanagerURL string
}

const dappmanagerURL = "http://dappmanager.dappnode:7000"

func StakerConfigForNetwork(pkg Pkg) StakerConfig {
	// Only hoodi network is supported
	network := "hoodi"
	execClients := []string{"hoodi-reth.dnp.dappnode.eth", "hoodi-geth.dnp.dappnode.eth", "hoodi-besu.dnp.dappnode.eth", "hoodi-erigon.dnp.dappnode.eth", "hoodi-nethermind.dnp.dappnode.eth"}
	consClients := []string{"prysm-hoodi.dnp.dappnode.eth", "teku-hoodi.dnp.dappnode.eth", "nimbus-hoodi.dnp.dappnode.eth", "lodestar-hoodi.dnp.dappnode.eth"}
	web3signer := "web3signer-hoodi.dnp.dappnode.eth"
	mevboost := "mev-boost-hoodi.dnp.dappnode.eth"
	relays := []string{}
	urls := Urls{
		ExecutionURL:   "http://execution.hoodi.dncore.dappnode:8545",
		BrainURL:       "http://brain.web3signer-hoodi.dappnode:5000",
		BeaconchainURL: "http://beacon-chain.hoodi.dncore.dappnode:3500",
		DappmanagerURL: dappmanagerURL,
	}

	ecDnpName := matchOrRandom(pkg.DnpName, execClients)
	ccDnpName := matchOrRandom(pkg.DnpName, consClients)

	return StakerConfig{
		ExecutionDnpName:         ecDnpName,
		ConsensusDnpName:         ccDnpName,
		Web3SignerDnpName:        web3signer,
		MevBoostDnpName:          mevboost,
		Relays:                   relays,
		Network:                  network,
		Urls:                     urls,
		BrainContainerName:       containerName("brain", web3signer),
		SignerContainerName:      containerName("web3signer", web3signer),
		BeaconchainContainerName: containerName("beacon-chain", ccDnpName),
		ValidatorContainerName:   containerName("validator", ccDnpName),
		ExecutionContainerName:   containerName(serviceNameFromExecutionClient(ecDnpName, network), ecDnpName),
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
