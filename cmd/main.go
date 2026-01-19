package main

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/composite"
	"clients-test/internal/application/domain"
	"clients-test/internal/application/services"
	"clients-test/internal/config"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")

	// Parse and validate configuration
	cfg := config.ParseConfig()
	cfg.Validate()

	// Set up Ctrl+C (SIGINT) handler to call composite cleaner
	cleanupDone := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx := context.Background()

	// Fetch dnpName from ipfs hash
	ipfsAdapter := ipfs.NewIPFSAdapter(&cfg.IPFSGatewayURL)
	pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, cfg.IPFSHash)
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to get dnpName from IPFS hash: %v", err)
	}

	// Retrieve staker config based on pkg (dnpName and serviceName)
	stakerConfig := domain.StakerConfigForNetwork(pkg)

	// print the staker config for debugging with each item on a new line
	printStakerConfig(logPrefix, stakerConfig)

	// Initialize API adapters
	snapshotsAdapter := snapshots.NewSnapshotsAdapter()
	dappManagerAdapter := dappmanager.NewDappManagerAdapter()
	brainAdapter := brain.NewBrainAdapter(stakerConfig.Urls.BrainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(stakerConfig.Urls.BeaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(stakerConfig.Urls.ExecutionURL)
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
	}

	// Initialize GitHub adapter for PR commenting
	githubAdapter := github.NewGitHubAdapter(cfg.GitHub)

	// Log GitHub configuration status
	if githubAdapter.IsEnabled() {
		logger.InfoWithPrefix(logPrefix, "GitHub integration enabled - will comment on PR #%d", cfg.GitHub.PRNumber)
	} else {
		logger.InfoWithPrefix(logPrefix, "GitHub integration not enabled (missing token, repository, or PR number)")
	}

	// Initialize the unified test adapter (now also initializes composites internally)
	compositeAdapter := composite.NewCompositeAdapter(
		dappManagerAdapter,
		brainAdapter,
		dockerAdapter,
		snapshotsAdapter,
		beaconchainAdapter,
		executionAdapter,
		ipfsAdapter,
		githubAdapter,
	)

	// Ctrl+C handler: call CleanEnvironment on composite
	go func() {
		sig := <-sigs
		logger.InfoWithPrefix(logPrefix, "Received signal: %v, running cleanup...", sig)
		err := compositeAdapter.CleanEnvironment(ctx, stakerConfig)
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

	if err := testRunner.RunTest(ctx, stakerConfig, pkg); err != nil {
		logger.FatalWithPrefix(logPrefix, "Test run failed: %v", err)
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
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
  ExecutionClientShortName: %s
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
		sc.ExecutionClientShortName,
		sc.Urls.ExecutionURL,
		sc.Urls.BrainURL,
		sc.Urls.BeaconchainURL,
		sc.Urls.DappmanagerURL,
		sc.Relays,
	)
	logger.InfoWithPrefix(prefix, "%s", msg)
}
