package download

import (
	"clients-test/internal/application/domain"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DownloadAdapter handles the .download_in_progress file operations
type DownloadAdapter struct {
	basePath string
}

// NewDownloadAdapter creates a new ProgressAdapter
func NewDownloadAdapter() *DownloadAdapter {
	return &DownloadAdapter{
		basePath: domain.SnapshotProgressPath,
	}
}

// NewDownloadAdapterWithPath creates a new DownloadAdapter with a custom base path (for testing)
func NewDownloadAdapterWithPath(basePath string) *DownloadAdapter {
	return &DownloadAdapter{
		basePath: basePath,
	}
}

// progressFilePath returns the full path to the progress file
func (p *DownloadAdapter) progressFilePath() string {
	return filepath.Join(p.basePath, domain.ProgressFileName)
}

// SetDownloadInProgress creates the .download_in_progress file
func (p *DownloadAdapter) SetDownloadInProgress(ctx context.Context) error {
	filePath := p.progressFilePath()

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
func (p *DownloadAdapter) ClearDownloadInProgress(ctx context.Context) error {
	filePath := p.progressFilePath()

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove progress file: %w", err)
	}

	return nil
}

// IsDownloadInProgress checks if download is in progress
func (p *DownloadAdapter) IsDownloadInProgress(ctx context.Context) (bool, error) {
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
func (p *DownloadAdapter) WaitForDownloadComplete(ctx context.Context, checkInterval time.Duration) error {
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
				return nil
			}

			time.Sleep(checkInterval)
		}
	}
}
