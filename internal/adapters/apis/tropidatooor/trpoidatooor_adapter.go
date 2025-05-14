package tropidatooor

import "net/http"

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
