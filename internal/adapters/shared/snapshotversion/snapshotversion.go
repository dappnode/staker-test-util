package snapshotversion

import (
	"clients-test/internal/application/domain"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Adapter handles reading the snapshot version file
type Adapter struct {
	basePath string
}

// NewAdapterWithPath creates a new Adapter with a custom base path (for testing or custom location)
func NewAdapterWithPath(basePath string) *Adapter {
	return &Adapter{basePath: basePath}
}

// snapshotVersionFilePath returns the full path to the snapshot version file
func (a *Adapter) snapshotVersionFilePath() string {
	return filepath.Join(a.basePath, domain.SnapshotVersionFileName)
}

// versionFileJSON is the structure of the JSON file
type versionFileJSON struct {
	Result string `json:"result"`
}

// GetSnapshotVersion reads the snapshot version file and returns the version string (the "result" field)
func (a *Adapter) GetSnapshotVersion(ctx context.Context) (string, error) {
	filePath := a.snapshotVersionFilePath()

	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open snapshot version file: %w", err)
	}
	defer f.Close()

	var data versionFileJSON
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode snapshot version file: %w", err)
	}

	if data.Result == "" {
		return "", fmt.Errorf("result field is empty in snapshot version file")
	}

	return data.Result, nil
}
