package main

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	cleaner "clients-test/internal/adapters/composite/envCleaner"
	ensurer "clients-test/internal/adapters/composite/envEnsurer"
	config "clients-test/internal/adapters/composite/envGetter"
	executor "clients-test/internal/adapters/composite/testExecutor"
	"clients-test/internal/adapters/system/mount"
	"clients-test/internal/application/services"
	"clients-test/internal/logger"
	"context"
	"os"
)

// TODO: intiialize ipfs before and fetch dnpName and then get the urls fr

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")

	ctx := context.Background()

	// Load config from env or flags (example IPFS gateway, base URLs)
	ipfsGateway := os.Getenv("IPFS_GATEWAY")
	if ipfsGateway == "" {
		ipfsGateway = "http://localhost:8080" // default
	}
	brainURL := os.Getenv("BRAIN_URL")
	if brainURL == "" {
		brainURL = "http://localhost:8081" // default
	}
	beaconchainURL := os.Getenv("BEACONCHAIN_URL")
	if beaconchainURL == "" {
		beaconchainURL = "http://localhost:5052" // default
	}
	dappmanagerURL := os.Getenv("DAPPMANAGER_URL")
	if dappmanagerURL == "" {
		dappmanagerURL = "http://localhost:8082" // default
	}
	executionURL := os.Getenv("EXECUTION_URL")
	if executionURL == "" {
		executionURL = "http://localhost:8545" // default
	}
	tropidatooorURL := os.Getenv("TROPIDATOOOR_URL")
	if tropidatooorURL == "" {
		tropidatooorURL = "http://localhost:8090" // default
	}

	// Initialize API adapters
	dappManagerAdapter := dappmanager.NewDappManagerAdapter(dappmanagerURL)
	brainAdapter := brain.NewBrainAdapter(brainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(beaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(executionURL)
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.ErrorWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
		os.Exit(1)
	}
	ipfsAdapter := ipfs.NewIPFSAdapter(ipfsGateway)
	tropidatooorAdapter := tropidatooor.NewTropidatooorAdapter(tropidatooorURL)
	mountAdapter := mount.NewMountAdapter()

	// Initialize composite adapters
	getter := config.NewConfigCompositeAdapter(brainAdapter, beaconchainAdapter, ipfsAdapter, tropidatooorAdapter)
	envEnsurer := ensurer.NewEnvEnsurerAdapter(dappManagerAdapter, brainAdapter, tropidatooorAdapter, dockerAdapter, mountAdapter, beaconchainAdapter, executionAdapter, ipfsAdapter)
	testExecutor := executor.NewTestExecutorAdapter(executionAdapter, brainAdapter, beaconchainAdapter)
	envCleaner := cleaner.NewEnvCleanerAdapter(dockerAdapter, mountAdapter)

	_ = getter // avoid unused variable warning

	// Initialize and run the service
	testRunner := services.NewTestRunner(envEnsurer, testExecutor, envCleaner)

	// Get IPFS hash from env or args (for demo, use env)
	ipfsHash := os.Getenv("IPFS_HASH")
	if ipfsHash == "" {
		logger.ErrorWithPrefix(logPrefix, "IPFS_HASH env var required")
		os.Exit(1)
	}

	if err := testRunner.RunTest(ctx, ipfsHash); err != nil {
		logger.ErrorWithPrefix(logPrefix, "Test run failed: %v", err)
		os.Exit(1)
	}

	logger.InfoWithPrefix(logPrefix, "Test run completed successfully")
}
