package main

import (
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/composite/snapshotmanager"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/adapters/shared/download"
	"clients-test/internal/application/domain"
	"clients-test/internal/application/services"
	"clients-test/internal/config"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
)

var logPrefix = "SNAPSHOT_CHECKER"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Snapshot Checker service")

	// Parse and validate configuration
	cfg := config.ParseSnapshotCheckerConfig()
	cfg.Validate()

	// Initialize domain config directly
	snapshotConfig := domain.SnapshotCheckerConfig{
		ExecutionClients: domain.GetExecutionClients(cfg.Network, cfg.ExecutionClients),
		CronIntervalSec:  cfg.CronIntervalSec,
		Network:          cfg.Network,
	}

	// Print configuration
	printSnapshotCheckerConfig(logPrefix, snapshotConfig)

	// Set up Ctrl+C (SIGINT) handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Initialize adapters
	snapshotsAdapter := snapshots.NewSnapshotsAdapter()
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Failed to init DockerAdapter: %v", err)
	}
	downloadAdapter := download.NewDownloadAdapter()
	blockNumberAdapter := blocknumber.NewBlockNumberAdapter()

	// Create composite snapshot manager adapter
	snapshotManagerAdapter := snapshotmanager.NewSnapshotManagerAdapter(
		snapshotsAdapter,
		dockerAdapter,
		blockNumberAdapter,
	)

	// If a previous run crashed/was interrupted, the marker file can be left behind and block future runs.
	// Clear it on startup so the service can recover.
	if inProgress, err := downloadAdapter.IsDownloadInProgress(ctx); err != nil {
		logger.WarnWithPrefix(logPrefix, "Failed to check %s marker: %v", domain.ProgressFileName, err)
	} else if inProgress {
		logger.WarnWithPrefix(logPrefix, "Found stale %s marker; clearing it on startup", domain.ProgressFileName)
		if err := downloadAdapter.ClearDownloadInProgress(context.Background()); err != nil {
			logger.WarnWithPrefix(logPrefix, "Failed to clear %s marker on startup: %v", domain.ProgressFileName, err)
		}
	}

	// Initialize the snapshot checker service
	service := services.NewSnapshotCheckerService(
		snapshotManagerAdapter,
		downloadAdapter,
		snapshotConfig,
	)

	go func() {
		sig := <-sigs
		logger.InfoWithPrefix(logPrefix, "Received signal: %v, shutting down...", sig)
		// Stop any running download containers using the service method
		service.StopAllDownloads(context.Background())
		// Best-effort cleanup: clear marker file so next run isn't blocked.
		if err := downloadAdapter.ClearDownloadInProgress(context.Background()); err != nil {
			logger.WarnWithPrefix(logPrefix, "Failed to clear %s marker on shutdown: %v", domain.ProgressFileName, err)
		}
		cancel()
	}()

	if err := service.Start(ctx, cfg.RunOnce); err != nil {
		if err == context.Canceled {
			logger.InfoWithPrefix(logPrefix, "Snapshot checker stopped gracefully")
		} else {
			logger.FatalWithPrefix(logPrefix, "Snapshot checker failed: %v", err)
		}
	}

	logger.InfoWithPrefix(logPrefix, "Snapshot checker service stopped")
}

// helper to pretty print snapshot checker config
func printSnapshotCheckerConfig(prefix string, sc domain.SnapshotCheckerConfig) {
	clients := append([]domain.ExecutionClientInfo(nil), sc.ExecutionClients...)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ShortName < clients[j].ShortName
	})

	var b strings.Builder
	b.WriteString("SnapshotCheckerConfig:\n")
	b.WriteString(fmt.Sprintf("  Network: %s\n", sc.Network))
	b.WriteString(fmt.Sprintf("  CronIntervalSec: %d\n", sc.CronIntervalSec))
	b.WriteString("  ExecutionClients:\n")
	for _, c := range clients {
		b.WriteString(fmt.Sprintf("    - ShortName: %s\n", c.ShortName))
		b.WriteString(fmt.Sprintf("      DnpName: %s\n", c.DnpName))
		b.WriteString(fmt.Sprintf("      VolumeName: %s\n", c.VolumeName))
		b.WriteString(fmt.Sprintf("      ContainerName: %s\n", c.ContainerName))
		b.WriteString(fmt.Sprintf("      VolumeTargetPath: %s\n", c.VolumeTargetPath))
	}

	msg := b.String()
	logger.InfoWithPrefix(prefix, "%s", msg)
}
