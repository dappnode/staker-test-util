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
	"clients-test/internal/logger"
)

var logPrefix = "Ensurer"

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

	// Log the timing
	if success {
		logger.InfoWithPrefix(logPrefix, "%s completed in %s", operationName, duration.Round(time.Millisecond))
	} else {
		logger.ErrorWithPrefix(logPrefix, "%s failed after %s: %v", operationName, duration.Round(time.Millisecond), err)
	}

	return err
}

// EnsureEnvironment validates the environment and prepares it for testing.
// It sets the staker config, installs the package, stops the container, downloads the snapshot, and starts the container.
// All operations are timed and recorded in the report.
func (e *EnsurerAdapter) EnsureEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg, report *domain.TestReport) error {
	var volumeTarget string

	// SetStakerConfig
	if err := timeOperation(report, "SetStakerConfig", func() error {
		return e.DappManager.SetStakerConfig(ctx, stakerConfig)
	}); err != nil {
		return fmt.Errorf("failed to set staker config for DNP: %w", err)
	}

	// PackageInstall
	if err := timeOperation(report, "PackageInstall", func() error {
		return e.DappManager.PackageInstall(ctx, pkg)
	}); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// StopAndGetVolumeTarget
	if err := timeOperation(report, "StopAndGetVolumeTarget", func() error {
		var err error
		volumeTarget, err = e.Docker.StopAndGetVolumeTarget(ctx, stakerConfig.ExecutionContainerName, stakerConfig.ExecutionVolumeName)
		return err
	}); err != nil {
		return fmt.Errorf("failed to stop container and get volume: %w", err)
	}

	// DownloadAndExtractSnapshot
	if err := timeOperation(report, "DownloadAndExtractSnapshot", func() error {
		return e.Snapshots.DownloadAndExtract(ctx, stakerConfig.Network, stakerConfig.ExecutionClientShortName, volumeTarget)
	}); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	// StartContainer
	if err := timeOperation(report, "StartContainer", func() error {
		return e.Docker.StartContainer(ctx, stakerConfig.ExecutionContainerName)
	}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}
