package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://snapshots.ethpandaops.io"
)

type SnapshotsAdapter struct {
	baseURL string
}

func NewSnapshotsAdapter() *SnapshotsAdapter {
	return &SnapshotsAdapter{
		baseURL: defaultBaseURL,
	}
}

func NewSnapshotsAdapterWithURL(baseURL string) *SnapshotsAdapter {
	return &SnapshotsAdapter{
		baseURL: baseURL,
	}
}

// GetBaseURL returns the base URL of the snapshots API
func (s *SnapshotsAdapter) GetBaseURL() string {
	return s.baseURL
}

// GetLatestBlockNumber fetches the latest available block number for a given network and client
func (s *SnapshotsAdapter) GetLatestBlockNumber(ctx context.Context, network, client string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/latest", s.baseURL, network, client)

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
	return blockNumber, nil
}

// GetSnapshotURL returns the full URL for a snapshot file
func (s *SnapshotsAdapter) GetSnapshotURL(network, client, blockNumber string) string {
	return fmt.Sprintf("%s/%s/%s/%s/snapshot.tar.zst", s.baseURL, network, client, blockNumber)
}

// GetClientVersion fetches the client version used to generate the snapshot
// The version is retrieved from the _snapshot_web3_clientVersion.json file
func (s *SnapshotsAdapter) GetClientVersion(ctx context.Context, network, client, blockNumber string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/_snapshot_web3_clientVersion.json", s.baseURL, network, client, blockNumber)

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
