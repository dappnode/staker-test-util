package domain

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type StakerConfig struct {
	ExecutionDnpName          string   `json:"executionDnpName"`
	ConsensusDnpName          string   `json:"consensusDnpName"`
	Web3SignerDnpName         string   `json:"web3signerDnpName"`
	MevBoostDnpName           string   `json:"mevBoostDnpName"`
	Relays                    []string `json:"relays,omitempty"` // Optional, can be empty
	Network                   string   `json:"network"`          // The network this config is for (e.g., mainnet, gnosis, hoodi, lukso)
	Urls                      Urls
	BrainContainerName        string // The name of the brain container
	SignerContainerName       string // The name of the web3signer container
	BeaconchainContainerName  string // The name of the beaconchain container
	ValidatorContainerName    string // The name of the validator container
	ExecutionContainerName    string // The name of the container to mount the NFS volume to
	ExecutionVolumeTargetPath string // Path to the execution client's docker volume data
}

type Urls struct {
	ExecutionURL   string
	BrainURL       string
	BeaconchainURL string
	DappmanagerURL string
}

// ClientOverrides holds optional client override settings
type ClientOverrides struct {
	ExecutionClient string // Short name like "geth", "reth", etc.
	ConsensusClient string // Short name like "prysm", "teku", etc.
}

// ClientOverrideResult holds the result of applying overrides with any warnings
type ClientOverrideResult struct {
	ExecutionDnpName string
	ConsensusDnpName string
	Warnings         []string
}

const dappmanagerURL = "http://dappmanager.dappnode:7000"

// hoodi network client lists
var (
	hoodiExecClients = []string{"hoodi-reth.dnp.dappnode.eth", "hoodi-geth.dnp.dappnode.eth", "hoodi-besu.dnp.dappnode.eth", "hoodi-erigon.dnp.dappnode.eth", "hoodi-nethermind.dnp.dappnode.eth"}
	hoodiConsClients = []string{"prysm-hoodi.dnp.dappnode.eth", "teku-hoodi.dnp.dappnode.eth", "nimbus-hoodi.dnp.dappnode.eth", "lodestar-hoodi.dnp.dappnode.eth"}
)

// StakerConfigFromOverrides creates a StakerConfig using only the override values.
// Used in sync mode where no IPFS hash/package is provided.
// If overrides are empty, random clients will be selected.
func StakerConfigFromOverrides(overrides ClientOverrides) (StakerConfig, []string) {
	// Use an empty Pkg so resolveClientsWithOverrides will use overrides or random
	return StakerConfigForNetwork(Pkg{}, overrides)
}

func StakerConfigForNetwork(pkg Pkg, overrides ClientOverrides) (StakerConfig, []string) {
	// Only hoodi network is supported
	network := "hoodi"
	web3signer := "web3signer-hoodi.dnp.dappnode.eth"
	mevboost := "mev-boost-hoodi.dnp.dappnode.eth"
	relays := []string{}
	urls := Urls{
		ExecutionURL:   "http://execution.hoodi.dncore.dappnode:8545",
		BrainURL:       "http://brain.web3signer-hoodi.dappnode:5000",
		BeaconchainURL: "http://beacon-chain.hoodi.dncore.dappnode:3500",
		DappmanagerURL: dappmanagerURL,
	}

	// Resolve execution and consensus clients with override logic
	result := resolveClientsWithOverrides(pkg, overrides, hoodiExecClients, hoodiConsClients)

	ecDnpName := result.ExecutionDnpName
	ccDnpName := result.ConsensusDnpName

	serviceName := serviceNameFromExecutionClient(ecDnpName, network)
	volumeName := getExecutionVolumeName(ecDnpName, serviceName)

	return StakerConfig{
		ExecutionDnpName:          ecDnpName,
		ConsensusDnpName:          ccDnpName,
		Web3SignerDnpName:         web3signer,
		MevBoostDnpName:           mevboost,
		Relays:                    relays,
		Network:                   network,
		Urls:                      urls,
		BrainContainerName:        containerName("brain", web3signer),
		SignerContainerName:       containerName("web3signer", web3signer),
		BeaconchainContainerName:  containerName("beacon-chain", ccDnpName),
		ValidatorContainerName:    containerName("validator", ccDnpName),
		ExecutionContainerName:    containerName(serviceName, ecDnpName),
		ExecutionVolumeTargetPath: fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName),
	}, result.Warnings
}

// resolveClientsWithOverrides determines execution and consensus clients based on:
// 1. If pkg matches an execution/consensus client, use it (overriding any flag/env with warning)
// 2. Otherwise, use the override if provided
// 3. Otherwise, pick a random client
func resolveClientsWithOverrides(pkg Pkg, overrides ClientOverrides, execClients, consClients []string) ClientOverrideResult {
	result := ClientOverrideResult{}

	// Check if pkg is an execution client
	pkgMatchedExec := matchClient(pkg.DnpName, execClients)
	// Check if pkg is a consensus client
	pkgMatchedCons := matchClient(pkg.DnpName, consClients)

	// Resolve execution client
	if pkgMatchedExec != "" {
		// Pkg is an execution client - use it
		if overrides.ExecutionClient != "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Package '%s' is an execution client; ignoring --execution-client flag '%s'",
					pkg.DnpName, overrides.ExecutionClient))
		}
		result.ExecutionDnpName = pkgMatchedExec
	} else if overrides.ExecutionClient != "" {
		// Use override if provided
		matched := matchClientByShortName(overrides.ExecutionClient, execClients)
		if matched != "" {
			result.ExecutionDnpName = matched
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Unknown execution client '%s'; using random", overrides.ExecutionClient))
			result.ExecutionDnpName = randomClient(execClients)
		}
	} else {
		// Pick random
		result.ExecutionDnpName = randomClient(execClients)
	}

	// Resolve consensus client
	if pkgMatchedCons != "" {
		// Pkg is a consensus client - use it
		if overrides.ConsensusClient != "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Package '%s' is a consensus client; ignoring --consensus-client flag '%s'",
					pkg.DnpName, overrides.ConsensusClient))
		}
		result.ConsensusDnpName = pkgMatchedCons
	} else if overrides.ConsensusClient != "" {
		// Use override if provided
		matched := matchClientByShortName(overrides.ConsensusClient, consClients)
		if matched != "" {
			result.ConsensusDnpName = matched
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Unknown consensus client '%s'; using random", overrides.ConsensusClient))
			result.ConsensusDnpName = randomClient(consClients)
		}
	} else {
		// Pick random
		result.ConsensusDnpName = randomClient(consClients)
	}

	return result
}

// matchClient returns the matching client dnpName if found, empty string otherwise
func matchClient(dnpName string, candidates []string) string {
	for _, c := range candidates {
		if strings.Contains(dnpName, c) || strings.Contains(c, dnpName) {
			return c
		}
	}
	return ""
}

// matchClientByShortName matches a short name (e.g., "geth", "prysm") to a full dnpName
func matchClientByShortName(shortName string, candidates []string) string {
	shortName = strings.ToLower(shortName)
	for _, c := range candidates {
		if strings.Contains(strings.ToLower(c), shortName) {
			return c
		}
	}
	return ""
}

// randomClient picks a random client from the list
func randomClient(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return candidates[r.Intn(len(candidates))]
}

// getExecutionVolumeName returns the docker volume name for the execution client
// reth and geth use their service name as volume name, others use "data"
func getExecutionVolumeName(dnpName, serviceName string) string {
	var volumeArg string
	if serviceName == "geth" || serviceName == "reth" {
		volumeArg = serviceName
	} else {
		volumeArg = "data"
	}
	return composeVolumeName(dnpName, volumeArg)
}
