package main

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/adapters/system/mount"
	"clients-test/internal/application/domain"
	"clients-test/internal/application/services"
	"clients-test/internal/logger"
	"context"
	"flag"
	"os"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")

	// CLI flags
	ipfsGatewayUrl := flag.String("ipfs-gateway-url", "", "IPFS gateway URL (required)")
	tropidatooorUrl := flag.String("tropidatooor-url", "", "Tropidatooor API URL (required)")
	ipfsHash := flag.String("ipfs-hash", "", "IPFS hash for the test package (required)")
	flag.Parse()

	if *ipfsGatewayUrl == "" || *tropidatooorUrl == "" || *ipfsHash == "" {
		logger.ErrorWithPrefix(logPrefix, "All flags --ipfs-gateway-url, --tropidatooor-url, and --ipfs-hash are required.")
		os.Exit(1)
	}

	ctx := context.Background()

	// Fetch dnpName from ipfs hash
	ipfsAdapter := ipfs.NewIPFSAdapter(ipfsGatewayUrl)
	pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, *ipfsHash)

	// Retrieve staker config based on pkg (dnpName and serviceName)
	stakerConfig := domain.StakerConfigForNetwork(pkg)

	// Get mount path
	tropidatooorAdapter := tropidatooor.NewTropidatooorAdapter(*tropidatooorUrl)
	mountConfig, err := tropidatooorAdapter.GetMountPath(ctx)
	if err != nil {
		logger.ErrorWithPrefix(logPrefix, "Failed to get mount path: %v", err)
		os.Exit(1)
	}

	// Initialize API adapters
	mountAdapter := mount.NewMountAdapter()
	dappManagerAdapter := dappmanager.NewDappManagerAdapter()
	brainAdapter := brain.NewBrainAdapter(stakerConfig.Urls.BrainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(stakerConfig.Urls.BeaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(stakerConfig.Urls.ExecutionURL)
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.ErrorWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
		os.Exit(1)
	}

	// Initialize composite adapters
	envEnsurer := ensurer.NewEnvEnsurerAdapter(dappManagerAdapter, brainAdapter, tropidatooorAdapter, dockerAdapter, mountAdapter, beaconchainAdapter, executionAdapter, ipfsAdapter)
	testExecutor := executor.NewTestExecutorAdapter(executionAdapter, brainAdapter, beaconchainAdapter)
	envCleaner := cleaner.NewEnvCleanerAdapter(executionAdapter, brainAdapter, beaconchainAdapter)

	// Initialize and run the service
	testRunner := services.NewTestRunner(envEnsurer, testExecutor, envCleaner)

	if err := testRunner.RunTest(ctx, mountConfig, stakerConfig, pkg); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Test run failed: %v", err)
		os.Exit(1)
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
}
