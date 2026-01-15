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
	"sync"
	"syscall"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting staker clients test runner")
	if err := run(); err != nil {
		logger.ErrorWithPrefix(logPrefix, "%v", err)
		os.Exit(1)
	}
}

func run() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// CLI flags
	ipfsGatewayUrl := flag.String("ipfs-gateway-url", "", "IPFS gateway URL (required)")
	tropidatooorUrl := flag.String("tropidatooor-url", "", "Tropidatooor API URL (required)")
	ipfsHash := flag.String("ipfs-hash", "", "IPFS hash for the test package (required)")
	flag.Parse()

	if *ipfsGatewayUrl == "" || *tropidatooorUrl == "" || *ipfsHash == "" {
		return fmt.Errorf("all flags --ipfs-gateway-url, --tropidatooor-url, and --ipfs-hash are required")
	}

	// Ensure cleanup runs at most once. Guarded so it only runs after setup is ready.
	var (
		cleanupOnce    sync.Once
		cleanupErr     error
		compositeAdptr *composite.CompositeAdapter
		mountConfig    *domain.Mount
		stakerConfig   domain.StakerConfig
	)

	cleanup := func(reason string) error {
		if compositeAdptr == nil || mountConfig == nil {
			return nil
		}
		cleanupOnce.Do(func() {
			logger.InfoWithPrefix(logPrefix, "Running cleanup (%s)...", reason)
			cleanupErr = compositeAdptr.CleanEnvironment(ctx, stakerConfig, *mountConfig)
			if cleanupErr != nil {
				logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", cleanupErr)
				return
			}
			logger.InfoWithPrefix(logPrefix, "Cleanup completed successfully")
		})
		return cleanupErr
	}

	defer func() {
		cerr := cleanup("defer")
		if err == nil && cerr != nil {
			err = fmt.Errorf("cleanup failed: %w", cerr)
		}
	}()

	// Fetch dnpName from ipfs hash
	ipfsAdapter := ipfs.NewIPFSAdapter(ipfsGatewayUrl)
	pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, *ipfsHash)
	if err != nil {
		return fmt.Errorf("failed to get dnpName from IPFS hash: %w", err)
	}

	// Retrieve staker config based on pkg (dnpName and serviceName)
	stakerConfig = domain.StakerConfigForNetwork(pkg)

	// print the staker config for debugging with each item on a new line
	printStakerConfig(logPrefix, stakerConfig)

	// Get mount path
	tropidatooorAdapter := tropidatooor.NewTropidatooorAdapter(*tropidatooorUrl)
	mountConfig, err = tropidatooorAdapter.DataRequest(ctx, stakerConfig.DataBackendName)
	if err != nil {
		return fmt.Errorf("failed to get mount path: %w", err)
	}

	// Initialize API adapters
	mountAdapter := mount.NewMountAdapter()
	dappManagerAdapter := dappmanager.NewDappManagerAdapter()
	brainAdapter := brain.NewBrainAdapter(stakerConfig.Urls.BrainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(stakerConfig.Urls.BeaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(stakerConfig.Urls.ExecutionURL)
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		return fmt.Errorf("failed to init DockerAdapter: %w", err)
	}

	// Initialize the unified test adapter (now also initializes composites internally)
	compositeAdptr = composite.NewCompositeAdapter(
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
	testRunner := services.NewTestRunner(compositeAdptr)
	if err := testRunner.RunTest(ctx, *mountConfig, stakerConfig, pkg); err != nil {
		return err
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
	return nil
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
