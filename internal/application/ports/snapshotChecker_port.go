package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

// SnapshotManager defines the interface for snapshot management operations
// This is the port that the snapshot checker service uses to interact with snapshot infrastructure
type SnapshotManager interface {
	// NeedsSnapshotDownload determines if a snapshot needs to be downloaded for the given client
	// Returns true if no snapshot exists or if a newer snapshot is available
	NeedsSnapshotDownload(ctx context.Context, client domain.ExecutionClientInfo, latestBlockNumber string) (bool, error)

	// DownloadAndMountSnapshot performs the complete snapshot download and mount process
	DownloadAndMountSnapshot(ctx context.Context, network string, client domain.ExecutionClientInfo) error

	// GetLatestBlockNumber fetches the latest available block number for a client
	GetLatestBlockNumber(ctx context.Context, network, client string) (string, error)

	// StopAllDownloads stops all running snapshot download containers
	StopAllDownloads(ctx context.Context)
}

// DownloadProgress defines the interface for tracking download progress state
// Used to prevent concurrent downloads and recover from interrupted downloads
type DownloadProgress interface {
	// IsDownloadInProgress checks if a download is currently in progress
	IsDownloadInProgress(ctx context.Context) (bool, error)

	// SetDownloadInProgress marks that a download has started
	SetDownloadInProgress(ctx context.Context) error

	// ClearDownloadInProgress marks that a download has completed (success or failure)
	ClearDownloadInProgress(ctx context.Context) error
}
