package domain

import (
	"fmt"
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
	ExecutionVolumeName      string // The name of the volume to mount for execution client data
	ExecutionClientShortName string // Short name of the execution client (e.g., geth, nethermind, reth)
}

type Urls struct {
	ExecutionURL   string
	BrainURL       string
	BeaconchainURL string
	DappmanagerURL string
}

const dappmanagerURL = "http://dappmanager.dappnode:7000"

func StakerConfigForNetwork(pkg Pkg) StakerConfig {
	network := getNetworkFromDnpName(pkg.DnpName)
	var execClients, consClients []string
	var web3signer, mevboost string
	var relays []string = nil
	var urls Urls

	switch network {
	case "gnosis":
		execClients = []string{"nethermind-xdai.dnp.dappnode.eth", "gnosis-erigon.dnp.dappnode.eth"}
		consClients = []string{"lighthouse-gnosis.dnp.dappnode.eth", "teku-gnosis.dnp.dappnode.eth", "nimbus-gnosis.dnp.dappnode.eth", "lodestar-gnosis.dnp.dappnode.eth"}
		web3signer = "web3signer-hoodi.dnp.dappnode.eth"
		mevboost = "mev-boost-hoodi.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.gnosis.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer-gnosis.dappnode:5000",
			BeaconchainURL: "http://beacon-chain.gnosis.dncore.dappnode:3500",
			DappmanagerURL: dappmanagerURL,
		}
	case "mainnet":
		execClients = []string{"nethermind.public.dappnode.eth", "geth.dnp.dappnode.eth", "erigon.dnp.dappnode.eth", "reth.dnp.dappnode.eth", "besu.public.dappnode.eth"}
		consClients = []string{"lighthouse.dnp.dappnode.eth", "prysm.dnp.dappnode.eth", "lodestar.dnp.dappnode.eth", "nimbus.dnp.dappnode.eth", "teku.dnp.dappnode.eth"}
		web3signer = "web3signer.dnp.dappnode.eth"
		mevboost = "mev-boost.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.mainnet.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer.dappnode:5000",
			BeaconchainURL: "http://beacon-chain.mainnet.dncore.dappnode:3500",
			DappmanagerURL: dappmanagerURL,
		}
	case "lukso":
		execClients = []string{"lukso-geth.dnp.dappnode.eth"}
		consClients = []string{"prysm-lukso.dnp.dappnode.eth", "teku-luks.dnp.dappnode.eth"}
		web3signer = "web3signer-lukso.dnp.dappnode.eth"
		mevboost = "mev-boost-lukso.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.lukso.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer-lukso.dappnode:5000",
			BeaconchainURL: "http://beacon-chain.lukso.dncore.dappnode:3500",
			DappmanagerURL: dappmanagerURL,
		}
	case "hoodi":
		execClients = []string{"hoodi-reth.dnp.dappnode.eth", "hoodi-geth.dnp.dappnode.eth", "hoodi-besu.dnp.dappnode.eth", "hoodi-erigon.dnp.dappnode.eth", "hoodi-nethermind.dnp.dappnode.eth"}
		consClients = []string{"prysm-hoodi.dnp.dappnode.eth", "teku-hoodi.dnp.dappnode.eth", "nimbus-hoodi.dnp.dappnode.eth", "lodestar-hoodi.dnp.dappnode.eth"}
		web3signer = "web3signer-hoodi.dnp.dappnode.eth"
		mevboost = "mev-boost-hoodi.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.hoodi.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer-hoodi.dappnode:5000",
			BeaconchainURL: "http://beacon-chain.hoodi.dncore.dappnode:3500",
			DappmanagerURL: dappmanagerURL,
		}
	}

	ecDnpName := matchOrRandom(pkg.DnpName, execClients)
	ccDnpName := matchOrRandom(pkg.DnpName, consClients)

	// List of known execution client short names
	ecShortNames := []string{"geth", "nethermind", "erigon", "reth", "besu"}
	ecShortDnpName := "unknown"
	for _, short := range ecShortNames {
		if strings.Contains(ecDnpName, short) {
			ecShortDnpName = short
			break
		}
	}

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
		ExecutionContainerName:   containerName(serviceNameFromExecutionClient(ecDnpName), ecDnpName),
		ExecutionVolumeName:      composeVolumeName(pkg.DnpName, pkg.ComposeVolumeName),
		ExecutionClientShortName: ecShortDnpName,
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

// Utility to get the servicename from a execution client (this utility assumes the service name is the first part of the dnp name)
func serviceNameFromExecutionClient(dnpName string) string {
	parts := strings.Split(dnpName, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return dnpName
}

// Utility to get the container name from service and dnpName. append dnp or public suffix depending on original dnpName
func containerName(serviceName, dnpName string) string {
	return fmt.Sprintf("DAppNodePackage-%s.%s", serviceName, dnpName)
}

// Utility to get the docker volume name from dnpName and compose volume name
// i.e hoodi-nethermind.dnp.dappnode.eth -> hoodi-netherminddnpdappnodeeth_<composeVolumeName>
func composeVolumeName(dnpName, composeVolumeName string) string {
	return fmt.Sprintf("%s_%s", strings.ReplaceAll(dnpName, ".", ""), composeVolumeName)
}
