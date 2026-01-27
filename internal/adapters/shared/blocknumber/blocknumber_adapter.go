package blocknumber

import (
	"clients-test/internal/application/domain"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// BlockNumberAdapter handles the snapshot_block_number file operations
type BlockNumberAdapter struct {
	basePath string
}

// NewBlockNumberAdapterWithPath creates a new BlockNumberAdapter with a custom base path
func NewBlockNumberAdapterWithPath(basePath string) *BlockNumberAdapter {
	return &BlockNumberAdapter{
		basePath: basePath,
	}
}

// blockNumberFilePath returns the full path to the block number file
func (b *BlockNumberAdapter) blockNumberFilePath() string {
	return filepath.Join(b.basePath, domain.SnapshotBlockNumberFileName)
}

// WriteBlockNumber writes the block number to the snapshot_block_number file
func (b *BlockNumberAdapter) WriteBlockNumber(ctx context.Context, blockNumber string) error {
	filePath := b.blockNumberFilePath()

	// Ensure directory exists
	if err := os.MkdirAll(b.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", b.basePath, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create block number file: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock for writing
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock on block number file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	_, err = f.WriteString(blockNumber)
	if err != nil {
		return fmt.Errorf("failed to write to block number file: %w", err)
	}

	return nil
}

// ReadBlockNumber reads the block number from the snapshot_block_number file
// Returns empty string if file doesn't exist
func (b *BlockNumberAdapter) ReadBlockNumber(ctx context.Context) (string, error) {
	filePath := b.blockNumberFilePath()

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to open block number file: %w", err)
	}
	defer f.Close()

	// Acquire shared lock for reading
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return "", fmt.Errorf("failed to acquire lock on block number file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read block number file: %w", err)
	}

	blockNumber := strings.TrimSpace(string(data))
	return blockNumber, nil
}

// BlockNumberExists checks if a block number file exists
func (b *BlockNumberAdapter) BlockNumberExists(ctx context.Context) (bool, error) {
	filePath := b.blockNumberFilePath()

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open block number file: %w", err)
	}
	defer f.Close()

	// Acquire shared lock for reading
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return false, fmt.Errorf("failed to acquire lock on block number file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return true, nil
}

// CompareBlockNumbers compares two block numbers
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (b *BlockNumberAdapter) CompareBlockNumbers(a, blockB string) int {
	aInt, aErr := strconv.ParseInt(a, 10, 64)
	bInt, bErr := strconv.ParseInt(blockB, 10, 64)

	if aErr != nil || bErr != nil {
		// Fall back to string comparison if not valid integers
		if a < blockB {
			return -1
		} else if a > blockB {
			return 1
		}
		return 0
	}

	if aInt < bInt {
		return -1
	} else if aInt > bInt {
		return 1
	}
	return 0
}

// IsNewerSnapshot checks if the latest available block number is newer than the current one
func (b *BlockNumberAdapter) IsNewerSnapshot(ctx context.Context, latestBlockNumber string) (bool, error) {
	currentBlockNumber, err := b.ReadBlockNumber(ctx)
	if err != nil {
		return false, err
	}

	// If no current block number, we need to download
	if currentBlockNumber == "" {
		return true, nil
	}

	// Compare block numbers
	return b.CompareBlockNumbers(latestBlockNumber, currentBlockNumber) > 0, nil
}
