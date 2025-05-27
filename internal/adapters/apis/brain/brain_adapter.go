package brain

import (
	"clients-test/internal/logger"
	"context"
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

// GetValidatorsPubkeys fetches the validator public keys from the brain service with context
func (b *BrainAdapter) GetValidatorsPubkeys(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/v0/brain/validators?tag=solo&format=pubkey", b.brainUrl)
	logger.Debug("[BrainAdapter] GetValidatorsPubkeys: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("[BrainAdapter] GetValidatorsPubkeys: failed to create request: %v", err)
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		logger.Error("[BrainAdapter] GetValidatorsPubkeys: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[BrainAdapter] GetValidatorsPubkeys: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("brain service error: %s", resp.Status)
	}

	var tagValidators map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&tagValidators); err != nil {
		logger.Error("[BrainAdapter] GetValidatorsPubkeys: failed to decode response: %v", err)
		return nil, err
	}
	pubkeys, ok := tagValidators["solo"]
	if !ok {
		logger.Error("[BrainAdapter] GetValidatorsPubkeys: no 'solo' tag found in response")
		return nil, fmt.Errorf("no 'solo' tag found in response")
	}
	logger.Debug("[BrainAdapter] GetValidatorsPubkeys: pubkeys=%+v", pubkeys)
	return pubkeys, nil
}
