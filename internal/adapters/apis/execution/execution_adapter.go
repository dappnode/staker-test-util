package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ExecutionAdapter implements the ExecutionPort interface
// It interacts with an Ethereum execution client via JSON-RPC
// See: https://ethereum.github.io/execution-apis/api-documentation/
type ExecutionAdapter struct {
	baseURL string
	client  *http.Client
}

// NewExecutionAdapter creates a new ExecutionAdapter
func NewExecutionAdapter(baseURL string) *ExecutionAdapter {
	return &ExecutionAdapter{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// GetClientVersion retrieves the version of the execution client
// See: https://ethereum.github.io/execution-apis/api-documentation/ (web3_clientVersion)
func (e *ExecutionAdapter) GetClientVersion(ctx context.Context) (string, error) {
	url := e.baseURL

	// JSON-RPC request body for web3_clientVersion
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "web3_clientVersion",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web3_clientVersion failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", err
	}

	return rpcResp.Result, nil
}

// GetIsSyncing retrieves the syncing status from the execution client with context
func (e *ExecutionAdapter) GetIsSyncing(ctx context.Context) (bool, error) {
	url := e.baseURL
	// JSON-RPC request body for eth_syncing
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_syncing",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("eth_syncing failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return false, err
	}

	// Per spec, result is either false (not syncing) or an object (syncing)
	if b, ok := rpcResp.Result.(bool); ok {
		return b, nil
	}
	return true, nil
}

// GetLatestBlockNumber retrieves the latest block number from the execution client
func (e *ExecutionAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	url := e.baseURL
	// JSON-RPC request body for eth_blockNumber
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("eth_blockNumber failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return 0, err
	}

	var blockNumber uint64
	_, err = fmt.Sscanf(rpcResp.Result, "0x%x", &blockNumber)
	if err != nil {
		return 0, fmt.Errorf("invalid block number format: %w", err)
	}

	return blockNumber, nil
}
