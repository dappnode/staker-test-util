package ports

import "context"

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
