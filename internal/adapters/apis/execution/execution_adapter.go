package execution

import (
	"bytes"
	"clients-test/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ExecutionAdapter implements the ExecutionPort interface
// It interacts with an Ethereum execution client via JSON-RPC
// See: https://ethereum.github.io/execution-apis/api-documentation/
type ExecutionAdapter struct {
	baseURL   string
	client    *http.Client
	logPrefix string
}

// NewExecutionAdapter creates a new ExecutionAdapter
func NewExecutionAdapter(baseURL string) *ExecutionAdapter {
	return &ExecutionAdapter{
		baseURL:   baseURL,
		client:    &http.Client{},
		logPrefix: "ExecutionAdapter",
	}
}

// GetClientVersion retrieves the version of the execution client
// See: https://ethereum.github.io/execution-apis/api-documentation/ (web3_clientVersion)
func (e *ExecutionAdapter) GetClientVersion(ctx context.Context) (string, error) {
	url := e.baseURL
	logger.DebugWithPrefix(e.logPrefix, "GetClientVersion: url=%s", url)

	// JSON-RPC request body for web3_clientVersion
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "web3_clientVersion",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetClientVersion: failed to marshal body: %v", err)
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetClientVersion: failed to create request: %v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetClientVersion: request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(e.logPrefix, "GetClientVersion: non-200 status: %s", resp.Status)
		return "", fmt.Errorf("web3_clientVersion failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetClientVersion: failed to decode response: %v", err)
		return "", err
	}

	logger.DebugWithPrefix(e.logPrefix, "GetClientVersion: version=%s", rpcResp.Result)
	return rpcResp.Result, nil
}

// GetIsSyncing retrieves the syncing status from the execution client with context
func (e *ExecutionAdapter) GetIsSyncing(ctx context.Context) (bool, error) {
	url := e.baseURL
	logger.DebugWithPrefix(e.logPrefix, "GetIsSyncing: url=%s", url)
	// JSON-RPC request body for eth_syncing
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_syncing",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetIsSyncing: failed to marshal body: %v", err)
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetIsSyncing: failed to create request: %v", err)
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetIsSyncing: request failed: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(e.logPrefix, "GetIsSyncing: non-200 status: %s", resp.Status)
		return false, fmt.Errorf("eth_syncing failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		logger.ErrorWithPrefix(e.logPrefix, "GetIsSyncing: failed to decode response: %v", err)
		return false, err
	}

	logger.DebugWithPrefix(e.logPrefix, "GetIsSyncing: result=%+v", rpcResp.Result)
	// Per spec, result is either false (not syncing) or an object (syncing)
	if b, ok := rpcResp.Result.(bool); ok {
		return b, nil
	}
	return true, nil
}
