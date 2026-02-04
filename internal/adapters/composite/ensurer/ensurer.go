package ensurer

import (
	"context"
	"fmt"
	"time"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
)

type EnsurerAdapter struct {
	DappManager *dappmanager.DappManagerAdapter
	Brain       *brain.BrainAdapter
	Docker      *docker.DockerAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
	Execution   *execution.ExecutionAdapter
	Ipfs        *ipfs.IPFSAdapter
}

func NewEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, docker *docker.DockerAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter, ipfs *ipfs.IPFSAdapter) *EnsurerAdapter {
	return &EnsurerAdapter{
		DappManager: dappManager,
		Brain:       brain,
		Docker:      docker,
		Beaconchain: beaconchain,
		Execution:   execution,
		Ipfs:        ipfs,
	}
}

// timeOperation measures the duration of an operation and records it in the report
func timeOperation(report *domain.TestReport, operationName string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	success := err == nil
	report.AddEnsureTiming(operationName, duration, success, err)

	return err
}

// EnsureEnvironment validates the environment and prepares it for testing.
// It sets the staker config and installs the package.
// All operations are timed and recorded in the report.
func (e *EnsurerAdapter) EnsureEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg, report *domain.TestReport) error {
	// Determine what type of client is being tested
	isExecutionTest := pkg.DnpName == stakerConfig.ExecutionDnpName
	isConsensusTest := pkg.DnpName == stakerConfig.ConsensusDnpName

	if isExecutionTest {
		report.TestedClientType = "execution"
	} else if isConsensusTest {
		report.TestedClientType = "consensus"
	}

	// SetStakerConfig
	if err := timeOperation(report, "SetStakerConfig", func() error {
		return e.DappManager.SetStakerConfig(ctx, stakerConfig)
	}); err != nil {
		return fmt.Errorf("failed to set staker config for DNP: %w", err)
	}

	// Capture client version BEFORE install (if client is already running)
	if version, err := getClientVersionWithRetry(func() (string, error) { return e.Execution.GetClientVersion(ctx) }, "execution", "before install"); err == nil {
		report.ExecutionClientVersionBefore = version
	}
	if version, err := getClientVersionWithRetry(func() (string, error) { return e.Beaconchain.GetClientVersion(ctx) }, "consensus", "before install"); err == nil {
		report.ConsensusClientVersionBefore = version
	}

	e.captureDnpVersions(ctx, stakerConfig, report)

	// PackageInstall
	if err := timeOperation(report, "PackageInstall", func() error {
		return e.DappManager.PackageInstall(ctx, pkg)
	}); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// Capture client version AFTER install
	if version, err := getClientVersionWithRetry(func() (string, error) { return e.Execution.GetClientVersion(ctx) }, "execution", "after install"); err == nil {
		report.ExecutionClientVersionAfter = version
	}
	if version, err := getClientVersionWithRetry(func() (string, error) { return e.Beaconchain.GetClientVersion(ctx) }, "consensus", "after install"); err == nil {
		report.ConsensusClientVersionAfter = version
	}

	return nil
}

// captureDnpVersions retrieves the DNP versions from the DappManager and stores them in the report
func (e *EnsurerAdapter) captureDnpVersions(ctx context.Context, stakerConfig domain.StakerConfig, report *domain.TestReport) {
	config, err := e.DappManager.GetStakerConfig(ctx, stakerConfig.Network)
	if err != nil {
		logger.Warn("Failed to get staker config for DNP version capture: %v", err)
		return
	}
	report.Web3SignerDnpVersion = config.Web3Signer.Data.Manifest.Version
	// find the execution and consensus from staker config that matches in the config
	for _, exec := range config.ExecutionClients {
		if exec.DnpName == stakerConfig.ExecutionDnpName {
			report.ExecutionDnpVersion = exec.Data.Manifest.Version
			break
		}
	}
	for _, cons := range config.ConsensusClients {
		if cons.DnpName == stakerConfig.ConsensusDnpName {
			report.ConsensusDnpVersion = cons.Data.Manifest.Version
			break
		}
	}
	if config.MevBoost != nil && stakerConfig.MevBoostDnpName != "" {
		report.MevBoostDnpVersion = config.MevBoost.Data.Manifest.Version
	}
}

// getClientVersionWithRetry tries to get the client version up to 30 times with 3s sleep between attempts
// The API takes some time to be available after package installation, so retries help avoid transient errors
func getClientVersionWithRetry(getVersionFunc func() (string, error), clientType string, stage string) (string, error) {
	var version string
	var err error
	for i := 0; i < 30; i++ {
		version, err = getVersionFunc()
		if err == nil {
			return version, nil
		}
		logger.Warn("Failed to get %s client version %s (attempt %d/30): %v", clientType, stage, i+1, err)
		time.Sleep(3 * time.Second)
	}
	return "", err
}
