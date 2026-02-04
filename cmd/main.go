package main

import (
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/adapters/apis/ipfs"
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

var logPrefix = "TEST_RUNNER"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Test Runner service")

	// Parse and validate configuration
	cfg := config.ParseConfig()
	cfg.Validate()

	logger.InfoWithPrefix(logPrefix, "Running in %s mode", cfg.Mode)

	// Set up Ctrl+C (SIGINT) handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Build overrides from config
	overrides := domain.ClientOverrides{
		ExecutionClient: cfg.ExecutionClient,
		ConsensusClient: cfg.ConsensusClient,
	}

	ipfsAdapter := ipfs.NewIPFSAdapter(&cfg.IPFSGatewayURL)

	var pkg domain.Pkg
	var stakerConfig domain.StakerConfig
	var warnings []string

	if cfg.Mode.IsTest() {
		// Test mode: fetch package info from IPFS hash
		pkg, err := ipfsAdapter.GetDnpNameAndServiceName(ctx, cfg.IPFSHash)
		if err != nil {
			logger.FatalWithPrefix(logPrefix, "Failed to get dnpName from IPFS hash: %v", err)
		}
		stakerConfig, warnings = domain.StakerConfigForNetwork(pkg, overrides)
	} else {
		// Sync mode: use overrides only, no IPFS required
		logger.InfoWithPrefix(logPrefix, "Sync mode: configuring staker from EXECUTION_CLIENT and CONSENSUS_CLIENT")
		stakerConfig, warnings = domain.StakerConfigFromOverrides(overrides)
	}

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

	// Log GitHub configuration status
	if githubAdapter.IsEnabled() {
		logger.InfoWithPrefix(logPrefix, "GitHub integration enabled - will comment on PR #%d", cfg.GitHub.PRNumber)
	} else {
		logger.InfoWithPrefix(logPrefix, "GitHub integration not enabled (missing token, repository, or PR number)")
	}

	// Initialize the unified test adapter (now also initializes composites internally)
	testManager := composite.NewTestManagerAdapter(
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
		cancel()
	}()

	// Initialize and run the service
	testRunner := services.NewTestRunner(testManager)

	if err := testRunner.Run(ctx, cfg.Mode, stakerConfig, pkg); err != nil {
		logger.FatalWithPrefix(logPrefix, "Run failed: %v", err)
	}

	logger.InfoWithPrefix(logPrefix, "Run completed successfully in %s mode", cfg.Mode)
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
