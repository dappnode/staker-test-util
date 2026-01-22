package domain

import (
	"fmt"
)

// ExecutionClientInfo contains the information for an execution client
type ExecutionClientInfo struct {
	ShortName        string // geth, nethermind, reth, besu, erigon
	DnpName          string // hoodi-nethermind.dnp.dappnode.eth
	VolumeName       string // Docker volume name
	ContainerName    string // Docker container name
	VolumeTargetPath string // Path to the volume data (/var/lib/docker/volumes/...)
}

// SnapshotCheckerConfig contains the configuration for the snapshot checker
type SnapshotCheckerConfig struct {
	ExecutionClients []ExecutionClientInfo // List of execution clients to manage
	CronIntervalSec  int                   // Interval between snapshot checks in seconds (default 6 hours)
	Network          string                // Network name (e.g., hoodi)
}

// ValidExecutionClients contains all valid execution client short names for hoodi
var ValidExecutionClients = []string{"geth", "nethermind", "reth", "besu", "erigon"}

// SnapshotProgressPath is the directory for progress files
const SnapshotProgressPath = "/usr/src/dappnode/DNCORE"

// ProgressFileName is the name of the download in progress file
const ProgressFileName = ".download_in_progress"

// TestProgressFileName is the name of the test in progress file
const TestProgressFileName = ".test_in_progress"

// SnapshotBlockNumberFileName returns the snapshot block number file name
const SnapshotBlockNumberFileName = "snapshot_block_number"

// GetExecutionClients returns the execution client info for hoodi network
func GetExecutionClients(network string, selectedClients []string) []ExecutionClientInfo {
	allClients := map[string]string{
		"reth": "hoodi-reth.dnp.dappnode.eth",
		"geth": "hoodi-geth.dnp.dappnode.eth",
		// "besu":       "hoodi-besu.dnp.dappnode.eth",
		// "erigon":     "hoodi-erigon.dnp.dappnode.eth",
		"nethermind": "hoodi-nethermind.dnp.dappnode.eth",
	}

	// Filter to selected clients or return all
	var result []ExecutionClientInfo
	for shortName, dnpName := range allClients {
		if len(selectedClients) == 0 || containsString(selectedClients, shortName) {
			serviceName := serviceNameFromExecutionClient(dnpName, network)
			// reth and geth use different volume names than data
			// - reth: https://github.com/dappnode/DAppNodePackage-reth-generic/blob/0de584dafd9b07fe24090f7e1cb96aa7f9108769/docker-compose.yml#L18
			// - geth: https://github.com/dappnode/DAppNodePackage-geth-generic/blob/1b9ed0da445e8599e7182a083cb3ffc98bf6b289/package_variants/hoodi/docker-compose.yml#L16
			var volumeArg string
			if shortName == "geth" || shortName == "reth" {
				volumeArg = shortName
			} else {
				volumeArg = "data"
			}
			volumeName := composeVolumeName(dnpName, volumeArg)
			volumeTargetPath := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)

			result = append(result, ExecutionClientInfo{
				ShortName:        shortName,
				DnpName:          dnpName,
				VolumeName:       volumeName,
				ContainerName:    containerName(serviceName, dnpName),
				VolumeTargetPath: volumeTargetPath,
			})
		}
	}

	return result
}

// IsValidExecutionClient checks if a client name is valid
func IsValidExecutionClient(client string) bool {
	return containsString(ValidExecutionClients, client)
}

// containsString checks if a string slice contains a value
func containsString(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
