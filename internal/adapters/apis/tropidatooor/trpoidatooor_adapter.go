package tropidatooor

import (
	"clients-test/internal/application/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TropidatooorAdapter is the adapter to interact with the Tropidatooor API
type TropidatooorAdapter struct {
	baseURL string
	client  *http.Client
}

// NewTropidatooorAdapter creates a new TropidatooorAdapter
func NewTropidatooorAdapter(baseURL string) *TropidatooorAdapter {
	return &TropidatooorAdapter{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// DataRequest sends a request to the Tropidatooor API to request data for a specific backend
func (t *TropidatooorAdapter) DataRequest(ctx context.Context, backendName string) (*domain.Mount, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/data/request/%s", t.baseURL, backendName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to request data for %s: status %s", backendName, resp.Status)
	}

	var dataResponse domain.Mount
	if err := json.NewDecoder(resp.Body).Decode(&dataResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &dataResponse, nil
}

// DataRelease sends a request to the Tropidatooor API to release data for a specific uniqueId
func (t *TropidatooorAdapter) DataRelease(ctx context.Context, uniqueId string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/data/release/%s", t.baseURL, uniqueId), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to release data for %s: status %s", uniqueId, resp.Status)
	}

	return nil
}
