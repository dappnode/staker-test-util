package ports

import (
	"clients-test/internal/application/domain"
	"context"
)

// SnapshotManager defines the interface for snapshot management operations
// This is the port that the snapshot checker service uses to interact with snapshot infrastructure
type SnapshotManager interface {
	// DownloadAndMountSnapshot performs the complete snapshot download and mount process
	DownloadAndMountSnapshot(ctx context.Context, network string, client domain.ExecutionClientInfo) error

	// GetLatestBlockNumber fetches the latest available block number for a client
	GetLatestBlockNumber(ctx context.Context, network, client string) (uint64, error)

	// StopDownload stops the snapshot download container for a specific client
	StopDownload(ctx context.Context, clientShortName string)

	// ClearExecutionClientData removes existing data for the execution client
	ClearExecutionClientData(ctx context.Context, client domain.ExecutionClientInfo) error
}
