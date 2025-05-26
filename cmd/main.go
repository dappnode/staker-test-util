package main

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/adapters/composite"
	"clients-test/internal/adapters/system/mount"
	"clients-test/internal/application/domain"
	"clients-test/internal/application/services"
	"clients-test/internal/logger"
	"context"
	"flag"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")

	// TODO: allow to set execution and consensus clients through flags
	// CLI flags
	ipfsGatewayUrl := flag.String("ipfs-gateway-url", "", "IPFS gateway URL (required)")
	tropidatooorUrl := flag.String("tropidatooor-url", "", "Tropidatooor API URL (required)")
	ipfsHash := flag.String("ipfs-hash", "", "IPFS hash for the test package (required)")
	flag.Parse()

	if *ipfsGatewayUrl == "" || *tropidatooorUrl == "" || *ipfsHash == "" {
		logger.FatalWithPrefix(logPrefix, "All flags --ipfs-gateway-url, --tropidatooor-url, and --ipfs-hash are required.")
	}

	ctx := context.Background()

	// Fetch dnpName from ipfs hash
	ipfsAdapter := ipfs.NewIPFSAdapter(ipfsGatewayUrl)
	pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, *ipfsHash)
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to get dnpName from IPFS hash: %v", err)
	}

	// Retrieve staker config based on pkg (dnpName and serviceName)
	stakerConfig := domain.StakerConfigForNetwork(pkg)

	// print the staker config for debugging with each item on a new line
	printStakerConfig(logPrefix, stakerConfig)

	// Get mount path
	tropidatooorAdapter := tropidatooor.NewTropidatooorAdapter(*tropidatooorUrl)
	mountConfig, err := tropidatooorAdapter.DataRequest(ctx, stakerConfig.DataBackendName)
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to get mount path: %v", err)
	}

	// Initialize API adapters
	mountAdapter := mount.NewMountAdapter()
	dappManagerAdapter := dappmanager.NewDappManagerAdapter()
	brainAdapter := brain.NewBrainAdapter(stakerConfig.Urls.BrainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(stakerConfig.Urls.BeaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(stakerConfig.Urls.ExecutionURL)
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
	}

	// Initialize the unified test adapter (now also initializes composites internally)
	composite := composite.NewCompositeAdapter(
		dappManagerAdapter,
		brainAdapter,
		tropidatooorAdapter,
		dockerAdapter,
		mountAdapter,
		beaconchainAdapter,
		executionAdapter,
		ipfsAdapter,
	)

	// Initialize and run the service
	testRunner := services.NewTestRunner(composite)

	if err := testRunner.RunTest(ctx, *mountConfig, stakerConfig, pkg); err != nil {
		logger.FatalWithPrefix(logPrefix, "Test run failed: %v", err)
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
}

// helper to pretty print staker config
func printStakerConfig(prefix string, sc domain.StakerConfig) {
	logger.InfoWithPrefix(prefix, "StakerConfig:")
	logger.InfoWithPrefix(prefix, "  ExecutionDnpName: %s", sc.ExecutionDnpName)
	logger.InfoWithPrefix(prefix, "  ConsensusDnpName: %s", sc.ConsensusDnpName)
	logger.InfoWithPrefix(prefix, "  Web3SignerDnpName: %s", sc.Web3SignerDnpName)
	logger.InfoWithPrefix(prefix, "  MevBoostDnpName: %s", sc.MevBoostDnpName)
	logger.InfoWithPrefix(prefix, "  Network: %s", sc.Network)
	logger.InfoWithPrefix(prefix, "  ExecutionContainerName: %s", sc.ExecutionContainerName)
	logger.InfoWithPrefix(prefix, "  DataBackendName: %s", sc.DataBackendName)
	logger.InfoWithPrefix(prefix, "  Urls:")
	logger.InfoWithPrefix(prefix, "    ExecutionURL: %s", sc.Urls.ExecutionURL)
	logger.InfoWithPrefix(prefix, "    BrainURL: %s", sc.Urls.BrainURL)
	logger.InfoWithPrefix(prefix, "    BeaconchainURL: %s", sc.Urls.BeaconchainURL)
	logger.InfoWithPrefix(prefix, "    DappmanagerURL: %s", sc.Urls.DappmanagerURL)
	logger.InfoWithPrefix(prefix, "  Relays:")
	if len(sc.Relays) == 0 {
		logger.InfoWithPrefix(prefix, "    (none)")
	} else {
		for _, r := range sc.Relays {
			logger.InfoWithPrefix(prefix, "    - %s", r)
		}
	}
}
