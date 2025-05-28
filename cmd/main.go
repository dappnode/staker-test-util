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
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")

	// Set up Ctrl+C (SIGINT) handler to call composite cleaner
	cleanupDone := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

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
	compositeAdapter := composite.NewCompositeAdapter(
		dappManagerAdapter,
		brainAdapter,
		tropidatooorAdapter,
		dockerAdapter,
		mountAdapter,
		beaconchainAdapter,
		executionAdapter,
		ipfsAdapter,
	)

	// Ctrl+C handler: call CleanEnvironment on composite
	go func() {
		sig := <-sigs
		logger.InfoWithPrefix(logPrefix, "Received signal: %v, running cleanup...", sig)
		err := compositeAdapter.CleanEnvironment(ctx, stakerConfig, *mountConfig)
		if err != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", err)
		} else {
			logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		}
		close(cleanupDone)
		os.Exit(1)
	}()

	// Initialize and run the service
	testRunner := services.NewTestRunner(compositeAdapter)

	if err := testRunner.RunTest(ctx, *mountConfig, stakerConfig, pkg); err != nil {
		logger.FatalWithPrefix(logPrefix, "Test run failed: %v", err)
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
	// Wait for cleanup if triggered
	<-cleanupDone
}

// helper to pretty print staker config
func printStakerConfig(prefix string, sc domain.StakerConfig) {
	// aggregate all fields into one log message
	msg := fmt.Sprintf(`StakerConfig:
  ExecutionDnpName: %s
  ConsensusDnpName: %s
  Web3SignerDnpName: %s
  MevBoostDnpName: %s
  Network: %s
  ExecutionContainerName: %s
  DataBackendName: %s
  Urls:
    ExecutionURL: %s
    BrainURL: %s
    BeaconchainURL: %s
    DappmanagerURL: %s
  Relays: %v`,
		sc.ExecutionDnpName,
		sc.ConsensusDnpName,
		sc.Web3SignerDnpName,
		sc.MevBoostDnpName,
		sc.Network,
		sc.ExecutionContainerName,
		sc.DataBackendName,
		sc.Urls.ExecutionURL,
		sc.Urls.BrainURL,
		sc.Urls.BeaconchainURL,
		sc.Urls.DappmanagerURL,
		sc.Relays,
	)
	logger.InfoWithPrefix(prefix, "%s", msg)
}
