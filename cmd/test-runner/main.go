package main

import (
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/composite/testmanager"
	"clients-test/internal/adapters/shared/download"
	"clients-test/internal/adapters/shared/testing"
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

var logPrefix = "TEST_RUNNER"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Test Runner service")

	// Parse and validate configuration
	cfg := config.ParseConfig()
	cfg.Validate()

	// Set up Ctrl+C (SIGINT) handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Fetch dnpName from ipfs hash
	ipfsAdapter := ipfs.NewIPFSAdapter(&cfg.IPFSGatewayURL)
	pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, cfg.IPFSHash)
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to get dnpName from IPFS hash: %v", err)
	}

	// Retrieve staker config based on pkg (dnpName and serviceName) with optional overrides
	overrides := domain.ClientOverrides{
		ExecutionClient: cfg.ExecutionClient,
		ConsensusClient: cfg.ConsensusClient,
	}
	stakerConfig, warnings := domain.StakerConfigForNetwork(pkg, overrides)

	// Log any warnings from client resolution
	for _, warning := range warnings {
		logger.WarnWithPrefix(logPrefix, "%s", warning)
	}

	// print the staker config for debugging with each item on a new line
	printStakerConfig(logPrefix, stakerConfig)

	// Initialize API adapters

	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
	}

	// Initialize GitHub adapter for PR commenting
	githubAdapter := github.NewGitHubAdapter(cfg.GitHub)

	// Initialize shared adapters with execution client's volume path
	logger.InfoWithPrefix(logPrefix, "Using volume path for flag files: %s", stakerConfig.ExecutionVolumeTargetPath)
	downloadAdapter := download.NewDownloadAdapterWithPath(stakerConfig.ExecutionVolumeTargetPath)
	testAdapter := testing.NewTestAdapterWithPath(stakerConfig.ExecutionVolumeTargetPath)

	// Log GitHub configuration status
	if githubAdapter.IsEnabled() {
		logger.InfoWithPrefix(logPrefix, "GitHub integration enabled - will comment on PR #%d", cfg.GitHub.PRNumber)
	} else {
		logger.InfoWithPrefix(logPrefix, "GitHub integration not enabled (missing token, repository, or PR number)")
	}

	// Initialize the unified test adapter (now also initializes composites internally)
	testManager := testmanager.NewTestManagerAdapter(
		stakerConfig,
		dockerAdapter,
		ipfsAdapter,
		githubAdapter,
	)

	// Ctrl+C handler: call CleanEnvironment on testManager
	go func() {
		sig := <-sigs
		logger.InfoWithPrefix(logPrefix, "Received signal: %v, shutting down...", sig)
		err := testManager.CleanEnvironment(context.Background(), stakerConfig)
		if err != nil {
			logger.ErrorWithPrefix(logPrefix, "Cleanup failed: %v", err)
		}
		// Best-effort cleanup, clear marker file `.test_in_progress` so next run isn't blocked.
		if err := testAdapter.ClearTestInProgress(context.Background()); err != nil {
			logger.WarnWithPrefix(logPrefix, "Failed to clear %s marker on shutdown: %v", domain.TestProgressFileName, err)
		}
		cancel()
	}()

	// Initialize and run the service
	testRunner := services.NewTestRunner(testManager, downloadAdapter, testAdapter)

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
  ExecutionVolumeTargetPath: %s
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
		sc.ExecutionVolumeTargetPath,
		sc.Urls.ExecutionURL,
		sc.Urls.BrainURL,
		sc.Urls.BeaconchainURL,
		sc.Urls.DappmanagerURL,
		sc.Relays,
	)
	logger.InfoWithPrefix(prefix, "%s", msg)
}
