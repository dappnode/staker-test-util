package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

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

// GetClientVersion fetches the client version used to generate the snapshot
// The version is retrieved from the _snapshot_web3_clientVersion.json file
func (s *SnapshotsAdapter) GetClientVersion(ctx context.Context, network, client, blockNumber string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/_snapshot_web3_clientVersion.json", s.baseURL, network, client, blockNumber)
	logger.DebugWithPrefix(s.logPrefix, "Fetching client version from %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch client version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch client version: status %s", resp.Status)
	}

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode client version response: %w", err)
	}

	logger.DebugWithPrefix(s.logPrefix, "Client version for %s/%s/%s: %s", network, client, blockNumber, result.Result)
	return result.Result, nil
}

// GetLatestClientVersion fetches the client version for the latest available snapshot
func (s *SnapshotsAdapter) GetLatestClientVersion(ctx context.Context, network, client string) (string, error) {
	blockNumber, err := s.GetLatestBlockNumber(ctx, network, client)
	if err != nil {
		return "", fmt.Errorf("failed to get latest block number: %w", err)
	}
	return s.GetClientVersion(ctx, network, client, blockNumber)
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
	// Optimizations:
	// - zstd -T0: multi-threaded decompression using all CPU cores
	// - No -v flag on tar: avoid printing every filename (major speedup)
	// - Streaming: download and extraction happen in parallel via pipe
	shellScript := fmt.Sprintf(`
set -e
apk add --no-cache wget tar zstd
echo "Downloading snapshot for block number: %s"
echo "Using $(nproc) CPU cores for decompression"
wget --tries=0 --retry-connrefused -O - %s | zstd -d -T0 | tar -xf - -C /data
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

	start := time.Now()
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.ErrorWithPrefix(s.logPrefix, "Failed to download/extract snapshot: %v\n%s", err, string(output))
		return fmt.Errorf("failed to download/extract snapshot: %w\n%s", err, string(output))
	}
	elapsed := time.Since(start)

	logger.InfoWithPrefix(s.logPrefix, "Successfully downloaded and extracted snapshot to %s in %s", targetPath, elapsed)
	logger.DebugWithPrefix(s.logPrefix, "Docker output:\n%s", string(output))
	return nil
}
