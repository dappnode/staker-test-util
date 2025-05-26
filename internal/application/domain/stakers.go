package domain

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type StakerConfig struct {
	ExecutionDnpName       string   `json:"executionDnpName"`
	ConsensusDnpName       string   `json:"consensusDnpName"`
	Web3SignerDnpName      string   `json:"web3SignerDnpName"`
	MevBoostDnpName        string   `json:"mevBoostDnpName"`
	Relays                 []string `json:"relays,omitempty"` // Optional, can be empty
	Network                string   `json:"network"`          // The network this config is for (e.g., mainnet, gnosis, hoodi, lukso)
	Urls                   Urls
	ExecutionContainerName string // The name of the container to mount the NFS volume to
	DataBackendName        string // The name of the backend to use for data requests
}

type Urls struct {
	ExecutionURL   string
	BrainURL       string
	BeaconchainURL string
	DappmanagerURL string
}

func StakerConfigForNetwork(pkg Pkg) StakerConfig {
	network := getNetworkFromDnpName(pkg.DnpName)
	var execClients, consClients []string
	var web3signer, mevboost string
	var relays []string = nil
	var urls Urls
	var dataBackend string

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
			BeaconchainURL: "http://beaconchain.gnosis.dncore.dappnode:3500",
			DappmanagerURL: "http://dappmanager.dappnode:7000",
		}
		dataBackend = "nethermind-gnosis"
	case "mainnet":
		execClients = []string{"nethermind.public.dappnode.eth", "geth.dnp.dappnode.eth", "erigon.dnp.dappnode.eth", "reth.dnp.dappnode.eth", "besu.public.dappnode.eth"}
		consClients = []string{"lighthouse.dnp.dappnode.eth", "prysm.dnp.dappnode.eth", "lodestar.dnp.dappnode.eth", "nimbus.dnp.dappnode.eth", "teku.dnp.dappnode.eth"}
		web3signer = "web3signer.dnp.dappnode.eth"
		mevboost = "mev-boost.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.mainnet.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer.dappnode:5000",
			BeaconchainURL: "http://beaconchain.mainnet.dncore.dappnode:3500",
			DappmanagerURL: "http://dappmanager.dappnode:7000",
		}
		dataBackend = "geth-mainnet"
	case "lukso":
		execClients = []string{"lukso-geth.dnp.dappnode.eth"}
		consClients = []string{"prysm-lukso.dnp.dappnode.eth", "teku-luks.dnp.dappnode.eth"}
		web3signer = "web3signer-lukso.dnp.dappnode.eth"
		mevboost = "mev-boost-lukso.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.lukso.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer-lukso.dappnode:5000",
			BeaconchainURL: "http://beaconchain.lukso.dncore.dappnode:3500",
			DappmanagerURL: "http://dappmanager.dappnode:7000",
		}
		dataBackend = "geth-lukso"
	case "hoodi":
		execClients = []string{"hoodi-reth.dnp.dappnode.eth", "hoodi-geth.dnp.dappnode.eth", "hoodi-besu.dnp.dappnode.eth", "hoodi-erigon.dnp.dappnode.eth", "hoodi-nethermind.dnp.dappnode.eth"}
		consClients = []string{"prysm-hoodi.dnp.dappnode.eth", "teku-hoodi.dnp.dappnode.eth", "nimbus-hoodi.dnp.dappnode.eth", "lodestar-hoodi.dnp.dappnode.eth", "lighthouse-hoodi.dnp.dappnode.eth"}
		web3signer = "web3signer-hoodi.dnp.dappnode.eth"
		mevboost = "mev-boost-hoodi.dnp.dappnode.eth"
		relays = []string{}
		urls = Urls{
			ExecutionURL:   "http://execution.hoodi.dncore.dappnode:8545",
			BrainURL:       "http://brain.web3signer-hoodi.dappnode:5000",
			BeaconchainURL: "http://beaconchain.hoodi.dncore.dappnode:3500",
			DappmanagerURL: "http://dappmanager.dappnode:8080",
		}
		dataBackend = "geth-hoodi"
	}

	exec := matchOrRandom(pkg.DnpName, execClients)
	cons := matchOrRandom(pkg.DnpName, consClients)
	return StakerConfig{
		ExecutionDnpName:       exec,
		ConsensusDnpName:       cons,
		Web3SignerDnpName:      web3signer,
		MevBoostDnpName:        mevboost,
		Relays:                 relays,
		Network:                network,
		Urls:                   urls,
		ExecutionContainerName: executionContainerName(pkg.ServiceName, pkg.DnpName),
		DataBackendName:        dataBackend,
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
func executionContainerName(serviceName, dnpName string) string {
	return fmt.Sprintf("DAppNodePackage-%s.%s.dnp.dappnode.eth", serviceName, shortDnpName(dnpName))
}
