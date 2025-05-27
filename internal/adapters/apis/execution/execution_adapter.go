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

// GetIsSyncing retrieves the syncing status from the execution client with context
func (e *ExecutionAdapter) GetIsSyncing(ctx context.Context) (bool, error) {
	url := e.baseURL
	logger.Debug("[ExecutionAdapter] GetIsSyncing: url=%s", url)
	// JSON-RPC request body for eth_syncing
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_syncing",
		"params":  []interface{}{},
		"id":      1,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		logger.Error("[ExecutionAdapter] GetIsSyncing: failed to marshal body: %v", err)
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.Error("[ExecutionAdapter] GetIsSyncing: failed to create request: %v", err)
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		logger.Error("[ExecutionAdapter] GetIsSyncing: request failed: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[ExecutionAdapter] GetIsSyncing: non-200 status: %s", resp.Status)
		return false, fmt.Errorf("eth_syncing failed: %s", resp.Status)
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		logger.Error("[ExecutionAdapter] GetIsSyncing: failed to decode response: %v", err)
		return false, err
	}

	logger.Debug("[ExecutionAdapter] GetIsSyncing: result=%+v", rpcResp.Result)
	// Per spec, result is either false (not syncing) or an object (syncing)
	if b, ok := rpcResp.Result.(bool); ok {
		return b, nil
	}
	return true, nil
}
