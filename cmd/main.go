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
