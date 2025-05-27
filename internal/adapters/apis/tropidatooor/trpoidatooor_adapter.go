package tropidatooor

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TropidatooorAdapter is the adapter to interact with the Tropidatooor API
type TropidatooorAdapter struct {
	baseURL   string
	client    *http.Client
	logPrefix string
}

// NewTropidatooorAdapter creates a new TropidatooorAdapter
func NewTropidatooorAdapter(baseURL string) *TropidatooorAdapter {
	return &TropidatooorAdapter{
		baseURL:   baseURL,
		client:    &http.Client{},
		logPrefix: "TropidatooorAdapter",
	}
}

// DataRequest sends a request to the Tropidatooor API to request data for a specific backend
func (t *TropidatooorAdapter) DataRequest(ctx context.Context, backendName string) (*domain.Mount, error) {
	logger.DebugWithPrefix(t.logPrefix, "DataRequest: backendName=%s", backendName)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/data/request/%s", t.baseURL, backendName), nil)
	if err != nil {
		logger.ErrorWithPrefix(t.logPrefix, "DataRequest: failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(t.logPrefix, "DataRequest: failed to send request: %v", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(t.logPrefix, "DataRequest: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("failed to request data for %s: status %s", backendName, resp.Status)
	}

	var dataResponse domain.Mount
	if err := json.NewDecoder(resp.Body).Decode(&dataResponse); err != nil {
		logger.ErrorWithPrefix(t.logPrefix, "DataRequest: failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.DebugWithPrefix(t.logPrefix, "DataRequest: success, dataResponse=%+v", dataResponse)
	return &dataResponse, nil
}

// DataRelease sends a request to the Tropidatooor API to release data for a specific uniqueId
func (t *TropidatooorAdapter) DataRelease(ctx context.Context, uniqueId string) error {
	logger.DebugWithPrefix(t.logPrefix, "DataRelease: uniqueId=%s", uniqueId)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/data/release/%s", t.baseURL, uniqueId), nil)
	if err != nil {
		logger.ErrorWithPrefix(t.logPrefix, "DataRelease: failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(t.logPrefix, "DataRelease: failed to send request: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(t.logPrefix, "DataRelease: non-200 status: %s", resp.Status)
		return fmt.Errorf("failed to release data for %s: status %s", uniqueId, resp.Status)
	}

	logger.DebugWithPrefix(t.logPrefix, "DataRelease: success for uniqueId=%s", uniqueId)
	return nil
}
