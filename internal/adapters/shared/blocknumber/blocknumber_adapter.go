package blocknumber

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var logPrefix = "BlockNumberAdapter"

// BlockNumberAdapter handles the snapshot_block_number file operations
type BlockNumberAdapter struct{}

// NewBlockNumberAdapter creates a new BlockNumberAdapter
func NewBlockNumberAdapter() *BlockNumberAdapter {
	return &BlockNumberAdapter{}
}

// blockNumberFilePath returns the full path to the block number file inside a volume
func (b *BlockNumberAdapter) blockNumberFilePath(volumeTargetPath string) string {
	return filepath.Join(volumeTargetPath, domain.SnapshotBlockNumberFileName)
}

// WriteBlockNumber writes the block number to the snapshot_block_number file in the volume
func (b *BlockNumberAdapter) WriteBlockNumber(ctx context.Context, volumeTargetPath string, blockNumber string) error {
	filePath := b.blockNumberFilePath(volumeTargetPath)
	logger.InfoWithPrefix(logPrefix, "Writing block number %s to %s", blockNumber, filePath)

	// Ensure directory exists
	if err := os.MkdirAll(volumeTargetPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", volumeTargetPath, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create block number file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(blockNumber)
	if err != nil {
		return fmt.Errorf("failed to write to block number file: %w", err)
	}

	return nil
}

// ReadBlockNumber reads the block number from the snapshot_block_number file in the volume
// Returns empty string if file doesn't exist
func (b *BlockNumberAdapter) ReadBlockNumber(ctx context.Context, volumeTargetPath string) (string, error) {
	filePath := b.blockNumberFilePath(volumeTargetPath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.DebugWithPrefix(logPrefix, "Block number file does not exist at %s", filePath)
			return "", nil
		}
		return "", fmt.Errorf("failed to read block number file: %w", err)
	}

	blockNumber := strings.TrimSpace(string(data))
	logger.DebugWithPrefix(logPrefix, "Read block number %s from %s", blockNumber, filePath)
	return blockNumber, nil
}

// BlockNumberExists checks if a block number file exists in the volume
func (b *BlockNumberAdapter) BlockNumberExists(ctx context.Context, volumeTargetPath string) (bool, error) {
	filePath := b.blockNumberFilePath(volumeTargetPath)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check block number file: %w", err)
	}
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
func (b *BlockNumberAdapter) IsNewerSnapshot(ctx context.Context, volumeTargetPath string, latestBlockNumber string) (bool, error) {
	currentBlockNumber, err := b.ReadBlockNumber(ctx, volumeTargetPath)
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
