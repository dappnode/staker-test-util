package snapshots

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"clients-test/internal/logger"
)

const (
	defaultBaseURL = "https://snapshots.ethpandaops.io"
	alpineImage    = "alpine:latest"
)

type SnapshotsAdapter struct {
	baseURL   string
	logPrefix string
}

func NewSnapshotsAdapter() *SnapshotsAdapter {
	return &SnapshotsAdapter{
		baseURL:   defaultBaseURL,
		logPrefix: "SnapshotsAdapter",
	}
}

func NewSnapshotsAdapterWithURL(baseURL string) *SnapshotsAdapter {
	return &SnapshotsAdapter{
		baseURL:   baseURL,
		logPrefix: "SnapshotsAdapter",
	}
}

// GetLatestBlockNumber fetches the latest available block number for a given network and client
func (s *SnapshotsAdapter) GetLatestBlockNumber(ctx context.Context, network, client string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/latest", s.baseURL, network, client)
	logger.DebugWithPrefix(s.logPrefix, "Fetching latest block number from %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest block number: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest block number: status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	blockNumber := strings.TrimSpace(string(body))
	logger.DebugWithPrefix(s.logPrefix, "Latest block number for %s/%s: %s", network, client, blockNumber)
	return blockNumber, nil
}

// DownloadAndExtract downloads the snapshot for the given network/client and extracts it to targetPath
// It uses a Docker container with alpine to handle the download and extraction
func (s *SnapshotsAdapter) DownloadAndExtract(ctx context.Context, network, client, targetPath string) error {
	logger.InfoWithPrefix(s.logPrefix, "Starting snapshot download for %s/%s to %s", network, client, targetPath)

	// Get the latest block number first
	blockNumber, err := s.GetLatestBlockNumber(ctx, network, client)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}

	snapshotURL := fmt.Sprintf("%s/%s/%s/%s/snapshot.tar.zst", s.baseURL, network, client, blockNumber)
	logger.InfoWithPrefix(s.logPrefix, "Downloading snapshot from %s", snapshotURL)

	// Build the shell command to run inside the alpine container
	// This installs necessary tools, downloads the snapshot, and extracts it
	shellScript := fmt.Sprintf(`
set -e
apk add --no-cache wget curl tar zstd
echo "Downloading snapshot for block number: %s"
wget --tries=0 --retry-connrefused -O - %s | tar -I zstd -xvf - -C /data
echo "Snapshot extraction complete"
`, blockNumber, snapshotURL)

	// Run docker command
	// docker run --rm -v <targetPath>:/data alpine /bin/sh -c '<script>'
	dockerArgs := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/data", targetPath),
		"--entrypoint", "/bin/sh",
		alpineImage,
		"-c", shellScript,
	}

	logger.DebugWithPrefix(s.logPrefix, "Running docker command: docker %s", strings.Join(dockerArgs, " "))

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.ErrorWithPrefix(s.logPrefix, "Failed to download/extract snapshot: %v\n%s", err, string(output))
		return fmt.Errorf("failed to download/extract snapshot: %w\n%s", err, string(output))
	}

	logger.InfoWithPrefix(s.logPrefix, "Successfully downloaded and extracted snapshot to %s", targetPath)
	logger.DebugWithPrefix(s.logPrefix, "Docker output:\n%s", string(output))
	return nil
}

// GetSnapshotInfo returns information about the available snapshot for a network/client
type SnapshotInfo struct {
	Network     string
	Client      string
	BlockNumber string
	URL         string
}

func (s *SnapshotsAdapter) GetSnapshotInfo(ctx context.Context, network, client string) (*SnapshotInfo, error) {
	blockNumber, err := s.GetLatestBlockNumber(ctx, network, client)
	if err != nil {
		return nil, err
	}

	return &SnapshotInfo{
		Network:     network,
		Client:      client,
		BlockNumber: blockNumber,
		URL:         fmt.Sprintf("%s/%s/%s/%s/snapshot.tar.zst", s.baseURL, network, client, blockNumber),
	}, nil
}
