package testing

import (
	"clients-test/internal/application/domain"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// TestAdapter handles the .test_in_progress file operations
type TestAdapter struct {
	basePath string
}

// NewTestAdapterWithPath creates a new TestAdapter with a custom base path (for testing)
func NewTestAdapterWithPath(basePath string) *TestAdapter {
	return &TestAdapter{
		basePath: basePath,
	}
}

// progressFilePath returns the full path to the test progress file
func (t *TestAdapter) progressFilePath() string {
	return filepath.Join(t.basePath, domain.TestProgressFileName)
}

// SetTestInProgress creates the .test_in_progress file
func (t *TestAdapter) SetTestInProgress(ctx context.Context) error {
	filePath := t.progressFilePath()

	if err := os.MkdirAll(t.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", t.basePath, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create test progress file: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock for writing
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock on test progress file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	_, err = f.WriteString(time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to write to test progress file: %w", err)
	}

	return nil
}

// ClearTestInProgress removes the .test_in_progress file
func (t *TestAdapter) ClearTestInProgress(ctx context.Context) error {
	filePath := t.progressFilePath()

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove test progress file: %w", err)
	}

	return nil
}

// IsTestInProgress checks if a test is in progress
func (t *TestAdapter) IsTestInProgress(ctx context.Context) (bool, error) {
	filePath := t.progressFilePath()

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open test progress file: %w", err)
	}
	defer f.Close()

	// Acquire shared lock for reading
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return false, fmt.Errorf("failed to acquire lock on test progress file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return true, nil
}

// WaitForTestComplete waits until no test is in progress
// Returns an error if the context is cancelled or times out
func (t *TestAdapter) WaitForTestComplete(ctx context.Context, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			inProgress, err := t.IsTestInProgress(ctx)
			if err != nil {
				return fmt.Errorf("error checking test progress: %w", err)
			}

			if !inProgress {
				return nil
			}

			time.Sleep(checkInterval)
		}
	}
}
