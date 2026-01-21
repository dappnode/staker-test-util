package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"clients-test/internal/logger"
)

const (
	defaultBaseURL          = "https://snapshots.ethpandaops.io"
	alpineImage             = "alpine:latest"
	downloadContainerPrefix = "snapshot-download-"
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
	logger.InfoWithPrefix(s.logPrefix, "Downloading %s snapshot from %s", client, snapshotURL)

	// Build the shell command to run inside the alpine container
	// This installs necessary tools, downloads the snapshot, and extracts it
	// Optimizations:
	// - aria2c -x16 -s16: 16 parallel connections using HTTP range requests for faster download
	// - zstd -T0: multi-threaded decompression using all CPU cores
	// - No -v flag on tar: avoid printing every filename (major speedup)
	// Note: aria2c downloads to file first (can't stream with range requests), then we decompress
	// This requires ~2x disk space temporarily but is significantly faster on high-bandwidth connections
	shellScript := fmt.Sprintf(`
set -e
apk add --no-cache aria2 tar zstd pv bash > /dev/null 2>&1
echo "[%s] Downloading snapshot for block number: %s"
echo "[%s] Using 16 parallel connections with HTTP range requests"
aria2c -x16 -s16 --file-allocation=none --console-log-level=warn --summary-interval=30 --show-console-readout=false -d /data -o snapshot.tar.zst "%s" 2>&1 | awk '/^\[#/{print "[%s] " $0; fflush()}'
echo "[%s] Download complete. Extracting with $(nproc) CPU cores..."
bash -c 'pv -f -i 30 -N "%s" -ptebar /data/snapshot.tar.zst 2> >(tr "\r" "\n" >&2) | zstd -d -T0 | tar -xf - -C /data'
rm -f /data/snapshot.tar.zst
echo "[%s] Snapshot extraction complete"
`, client, blockNumber, client, snapshotURL, client, client, client, client)

	// Run docker command
	// docker run --rm --name <name> -v <targetPath>:/data alpine /bin/sh -c '<script>'
	// --name allows us to stop the container on shutdown
	// Note: we don't use -t (TTY) because multiple parallel downloads would overwrite each other's progress
	// Instead, pv with -f outputs line-based progress that works well in parallel
	containerName := fmt.Sprintf("%s%s", downloadContainerPrefix, client)
	dockerArgs := []string{
		"run", "--rm",
		"--name", containerName,
		"-v", fmt.Sprintf("%s:/data", targetPath),
		"--entrypoint", "/bin/sh",
		alpineImage,
		"-c", shellScript,
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	// Stream output in real-time instead of buffering until completion
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.ErrorWithPrefix(s.logPrefix, "Failed to download/extract snapshot: %v", err)
		return fmt.Errorf("failed to download/extract snapshot: %w", err)
	}
	elapsed := time.Since(start)

	logger.InfoWithPrefix(s.logPrefix, "Successfully downloaded and extracted snapshot to %s in %s", targetPath, elapsed)
	return nil
}

// StopAllDownloads stops all running snapshot download containers in parallel
func (s *SnapshotsAdapter) StopAllDownloads(ctx context.Context) {
	logger.InfoWithPrefix(s.logPrefix, "Stopping all snapshot download containers...")

	// Find all containers with our prefix
	listCmd := exec.CommandContext(ctx, "docker", "ps", "-q", "--filter", fmt.Sprintf("name=%s", downloadContainerPrefix))
	output, err := listCmd.Output()
	if err != nil {
		logger.WarnWithPrefix(s.logPrefix, "Failed to list download containers: %v", err)
		return
	}

	containerIDs := strings.TrimSpace(string(output))
	if containerIDs == "" {
		logger.DebugWithPrefix(s.logPrefix, "No download containers running")
		return
	}

	// Stop all containers in parallel
	ids := strings.Split(containerIDs, "\n")
	var wg sync.WaitGroup
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			logger.InfoWithPrefix(s.logPrefix, "Stopping download container %s", containerID)
			stopCmd := exec.Command("docker", "stop", "-t", "5", containerID)
			if err := stopCmd.Run(); err != nil {
				logger.WarnWithPrefix(s.logPrefix, "Failed to stop container %s: %v", containerID, err)
			}
		}(id)
	}
	wg.Wait()

	logger.InfoWithPrefix(s.logPrefix, "All download containers stopped")
}
