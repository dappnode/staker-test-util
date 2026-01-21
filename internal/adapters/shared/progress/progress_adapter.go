package progress

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logPrefix = "ProgressAdapter"

// ProgressAdapter handles the .download_in_progress file operations
type ProgressAdapter struct {
	basePath string
}

// NewProgressAdapter creates a new ProgressAdapter
func NewProgressAdapter() *ProgressAdapter {
	return &ProgressAdapter{
		basePath: domain.SnapshotProgressPath,
	}
}

// NewProgressAdapterWithPath creates a new ProgressAdapter with a custom base path (for testing)
func NewProgressAdapterWithPath(basePath string) *ProgressAdapter {
	return &ProgressAdapter{
		basePath: basePath,
	}
}

// progressFilePath returns the full path to the progress file
func (p *ProgressAdapter) progressFilePath() string {
	return filepath.Join(p.basePath, domain.ProgressFileName)
}

// SetDownloadInProgress creates the .download_in_progress file
func (p *ProgressAdapter) SetDownloadInProgress(ctx context.Context) error {
	filePath := p.progressFilePath()
	logger.InfoWithPrefix(logPrefix, "Setting download in progress at %s", filePath)

	if err := os.MkdirAll(p.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", p.basePath, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create progress file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to write to progress file: %w", err)
	}

	return nil
}

// ClearDownloadInProgress removes the .download_in_progress file
func (p *ProgressAdapter) ClearDownloadInProgress(ctx context.Context) error {
	filePath := p.progressFilePath()
	logger.InfoWithPrefix(logPrefix, "Clearing download in progress at %s", filePath)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove progress file: %w", err)
	}

	return nil
}

// IsDownloadInProgress checks if download is in progress
func (p *ProgressAdapter) IsDownloadInProgress(ctx context.Context) (bool, error) {
	filePath := p.progressFilePath()
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check progress file: %w", err)
	}
	return true, nil
}

// WaitForDownloadComplete waits until no download is in progress
// Returns an error if the context is cancelled or times out
func (p *ProgressAdapter) WaitForDownloadComplete(ctx context.Context, checkInterval time.Duration) error {
	logger.InfoWithPrefix(logPrefix, "Waiting for downloads to complete")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			inProgress, err := p.IsDownloadInProgress(ctx)
			if err != nil {
				return fmt.Errorf("error checking download progress: %w", err)
			}

			if !inProgress {
				logger.InfoWithPrefix(logPrefix, "No downloads in progress, continuing")
				return nil
			}

			logger.InfoWithPrefix(logPrefix, "Download in progress, waiting %v...", checkInterval)
			time.Sleep(checkInterval)
		}
	}
}
