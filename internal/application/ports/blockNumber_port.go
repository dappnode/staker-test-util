package ports

import "context"

// BlockNumber defines the interface for block number tracking operations
// Used to track the current snapshot block number and compare with available snapshots
type BlockNumber interface {
	// ReadBlockNumber reads the current block number from storage
	// Returns empty string if no block number file exists
	ReadBlockNumber(ctx context.Context) (string, error)

	// WriteBlockNumber writes the block number to storage
	WriteBlockNumber(ctx context.Context, blockNumber string) error

	// BlockNumberExists checks if a block number file exists
	BlockNumberExists(ctx context.Context) (bool, error)

	// IsNewerSnapshot checks if the latest available block number is newer than the current one
	IsNewerSnapshot(ctx context.Context, latestBlockNumber string) (bool, error)
}
