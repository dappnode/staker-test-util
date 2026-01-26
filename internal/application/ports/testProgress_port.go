package ports

import "context"

// TestProgress defines the interface for tracking test progress state
// Used to prevent snapshot downloads while a test is running
type TestProgress interface {
	// IsTestInProgress checks if a test is currently in progress
	IsTestInProgress(ctx context.Context) (bool, error)

	// SetTestInProgress marks that a test has started
	SetTestInProgress(ctx context.Context) error

	// ClearTestInProgress marks that a test has completed (success or failure)
	ClearTestInProgress(ctx context.Context) error
}
