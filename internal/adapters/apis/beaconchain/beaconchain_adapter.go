package beaconchain

import (
	"bytes"
	"clients-test/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BeaconchainAdapter implements the BeaconchainPort interface
// It interacts with an Ethereum beacon node via REST API
// See: https://ethereum.github.io/beacon-APIs/
type BeaconchainAdapter struct {
	beaconChainUrl string
	client         *http.Client
}

// NewBeaconchainAdapter creates a new BeaconchainAdapter
func NewBeaconchainAdapter(beaconChainUrl string) *BeaconchainAdapter {
	return &BeaconchainAdapter{
		beaconChainUrl: beaconChainUrl,
		client:         &http.Client{},
	}
}

// GetIsSyncing retrieves the syncing status from the beacon node with context
func (b *BeaconchainAdapter) GetIsSyncing(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/eth/v1/node/syncing", b.beaconChainUrl)
	logger.Debug("[BeaconchainAdapter] GetIsSyncing: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetIsSyncing: failed to create request: %v", err)
		return false, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetIsSyncing: request failed: %v", err)
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("[BeaconchainAdapter] GetIsSyncing: non-200 status: %s", resp.Status)
		return false, fmt.Errorf("beacon node syncing failed: %s", resp.Status)
	}
	var result struct {
		Data struct {
			IsSyncing bool `json:"is_syncing"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("[BeaconchainAdapter] GetIsSyncing: failed to decode response: %v", err)
		return false, err
	}
	logger.Debug("[BeaconchainAdapter] GetIsSyncing: result=%+v", result.Data.IsSyncing)
	return result.Data.IsSyncing, nil
}

// blockHeaderResponse for /eth/v1/beacon/headers/{block_id}
type blockHeaderResponse struct {
	Data struct {
		Header struct {
			Message struct {
				Slot string `json:"slot"`
			} `json:"message"`
		} `json:"header"`
	} `json:"data"`
}

// getBlockHeader retrieves the block header for a given block ID
func (b *BeaconchainAdapter) getBlockHeader(ctx context.Context, blockID string) (*blockHeaderResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/headers/%s", b.beaconChainUrl, blockID)
	logger.Debug("[BeaconchainAdapter] getBlockHeader: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("[BeaconchainAdapter] getBlockHeader: failed to create request: %v", err)
		return nil, fmt.Errorf("failed to send request to Beaconchain at %s: %w", url, err)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		logger.Error("[BeaconchainAdapter] getBlockHeader: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	var result blockHeaderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("[BeaconchainAdapter] getBlockHeader: failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response for GetBlockHeader: %w", err)
	}
	logger.Debug("[BeaconchainAdapter] getBlockHeader: slot=%s", result.Data.Header.Message.Slot)
	return &result, nil
}

func (b *BeaconchainAdapter) getEpochFinalized(ctx context.Context, blockID string) (uint64, error) {
	header, err := b.getBlockHeader(ctx, blockID)
	if err != nil {
		logger.Error("[BeaconchainAdapter] getEpochFinalized: failed to get block header for blockID %s: %v", blockID, err)
		return 0, fmt.Errorf("failed to get block header for blockID %s: %w", blockID, err)
	}
	slot := header.Data.Header.Message.Slot
	epoch := getEpochFromSlot(slot)
	logger.Debug("[BeaconchainAdapter] getEpochFinalized: slot=%s epoch=%d", slot, epoch)
	return epoch, nil
}

func getEpochFromSlot(slot string) uint64 {
	const slotsPerEpoch = 32
	slotInt := parseInt(slot)
	return slotInt / slotsPerEpoch
}

func parseInt(slot string) uint64 {
	var result uint64
	fmt.Sscanf(slot, "%d", &result)
	return result
}

// GetValidatorLiveness retrieves validator liveness for the current epoch and given validator indexes
func (b *BeaconchainAdapter) GetValidatorLiveness(ctx context.Context, indexes []string) (map[string]bool, error) {
	epoch, err := b.getEpochFinalized(ctx, "finalized")
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorLiveness: failed to get epoch finalized: %v", err)
		return nil, err
	}
	// Join indexes as comma-separated string
	joined := ""
	for i, idx := range indexes {
		if i > 0 {
			joined += ","
		}
		joined += idx
	}
	url := fmt.Sprintf("%s/eth/v1/validator/liveness/%d?indices=%s", b.beaconChainUrl, epoch, joined)
	logger.Debug("[BeaconchainAdapter] GetValidatorLiveness: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorLiveness: failed to create request: %v", err)
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorLiveness: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("[BeaconchainAdapter] GetValidatorLiveness: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("validator liveness failed: %s", resp.Status)
	}
	var result struct {
		Data []struct {
			Index  string `json:"index"`
			IsLive bool   `json:"is_live"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorLiveness: failed to decode response: %v", err)
		return nil, err
	}
	liveness := make(map[string]bool)
	for _, v := range result.Data {
		liveness[v.Index] = v.IsLive
	}
	logger.Debug("[BeaconchainAdapter] GetValidatorLiveness: liveness=%+v", liveness)
	return liveness, nil
}

// GetValidatorsIndexes retrieves the validator index for each given pubkey with status active_ongoing
func (b *BeaconchainAdapter) GetValidatorsIndexes(ctx context.Context, pubkeys []string) ([]string, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/states/finalized/validators", b.beaconChainUrl)
	logger.Debug("[BeaconchainAdapter] GetValidatorsIndexes: url=%s pubkeys=%+v", url, pubkeys)
	requestBody := struct {
		IDs      []string `json:"ids"`
		Statuses []string `json:"statuses"`
	}{
		IDs:      pubkeys,
		Statuses: []string{"active_ongoing"},
	}
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorsIndexes: failed to marshal request body: %v", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorsIndexes: failed to create request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorsIndexes: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[BeaconchainAdapter] GetValidatorsIndexes: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("get validators indexes failed: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			Index     string `json:"index"`
			Validator struct {
				Pubkey string `json:"pubkey"`
			} `json:"validator"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("[BeaconchainAdapter] GetValidatorsIndexes: failed to decode response: %v", err)
		return nil, err
	}
	indexes := make([]string, len(result.Data))
	for i, v := range result.Data {
		indexes[i] = v.Index
	}
	logger.Debug("[BeaconchainAdapter] GetValidatorsIndexes: indexes=%+v", indexes)
	return indexes, nil
}
