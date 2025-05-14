package tropidatooor

import (
	"context"
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

// GetMountPath retrieves the mount path from the Tropidatooor API
func (t *TropidatooorAdapter) GetMountPath(ctx context.Context) (string, error) {
	return "", nil
}
