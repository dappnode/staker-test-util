package brain

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BrainAdapter implements the BrainPort interface
// It interacts with the brain service to fetch validator indexes
// Example endpoint: /v0/brain/validators?tag=solo&format=index
type BrainAdapter struct {
	brainUrl string
	client   *http.Client
}

// NewBrainAdapter creates a new BrainAdapter
func NewBrainAdapter(brainUrl string) *BrainAdapter {
	return &BrainAdapter{
		brainUrl: brainUrl,
		client:   &http.Client{},
	}
}

// GetValidatorsPubkeys fetches the validator public keys from the brain service
func (b *BrainAdapter) GetValidatorsPubkeys() ([]string, error) {
	url := fmt.Sprintf("%s/v0/brain/validators?tag=solo&format=pubkey", b.brainUrl)
	resp, err := b.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brain service error: %s", resp.Status)
	}

	var tagValidators map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&tagValidators); err != nil {
		return nil, err
	}
	pubkeys, ok := tagValidators["solo"]
	if !ok {
		return nil, fmt.Errorf("no 'solo' tag found in response")
	}
	return pubkeys, nil
}
