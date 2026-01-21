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
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/application/domain"
)

type EnsurerAdapter struct {
	DappManager *dappmanager.DappManagerAdapter
	Brain       *brain.BrainAdapter
	Docker      *docker.DockerAdapter
	Snapshots   *snapshots.SnapshotsAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
	Execution   *execution.ExecutionAdapter
	Ipfs        *ipfs.IPFSAdapter
}

func NewEnsurerAdapter(dappManager *dappmanager.DappManagerAdapter, brain *brain.BrainAdapter, docker *docker.DockerAdapter, snapshotsAdapter *snapshots.SnapshotsAdapter, beaconchain *beaconchain.BeaconchainAdapter, execution *execution.ExecutionAdapter, ipfs *ipfs.IPFSAdapter) *EnsurerAdapter {
	return &EnsurerAdapter{
		DappManager: dappManager,
		Brain:       brain,
		Docker:      docker,
		Snapshots:   snapshotsAdapter,
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
	if isExecutionTest {
		if version, err := e.Execution.GetClientVersion(ctx); err == nil {
			report.ExecutionClientVersionBefore = version
		}
	} else if isConsensusTest {
		if version, err := e.Beaconchain.GetClientVersion(ctx); err == nil {
			report.ConsensusClientVersionBefore = version
		}
	}

	// PackageInstall
	if err := timeOperation(report, "PackageInstall", func() error {
		return e.DappManager.PackageInstall(ctx, pkg)
	}); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// Capture client version AFTER install
	if isExecutionTest {
		if version, err := e.Execution.GetClientVersion(ctx); err == nil {
			report.ExecutionClientVersionAfter = version
		}
	} else if isConsensusTest {
		if version, err := e.Beaconchain.GetClientVersion(ctx); err == nil {
			report.ConsensusClientVersionAfter = version
		}
	}

	return nil
}
