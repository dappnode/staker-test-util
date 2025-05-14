package tropidatooor

import (
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

// Ping sends a ping request to the Tropidatooor API with context
func (t *TropidatooorAdapter) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to ping Tropidatooor API: %s", resp.Status)
	}
	return nil
}

// GetMountPath retrieves the mount path and mount ID from the Tropidatooor API
func (t *TropidatooorAdapter) GetMountPath(ctx context.Context) (mountPath string, mountId string, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/mount-path", nil)
	if err != nil {
		return "", "", err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to get mount path from Tropidatooor API: %s", resp.Status)
	}
	var respObj struct {
		MountPath string `json:"mountPath"`
		MountId   string `json:"mountId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
		return "", "", err
	}
	return respObj.MountPath, respObj.MountId, nil
}
