package beaconchain

import (
	"bytes"
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

// GetIsSyncing retrieves the syncing status from the beacon node
func (b *BeaconchainAdapter) GetIsSyncing() (bool, error) {
	url := fmt.Sprintf("%s/eth/v1/node/syncing", b.beaconChainUrl)
	resp, err := b.client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("beacon node syncing failed: %s", resp.Status)
	}
	var result struct {
		Data struct {
			IsSyncing bool `json:"is_syncing"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
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
func (b *BeaconchainAdapter) getBlockHeader(blockID string) (*blockHeaderResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/headers/%s", b.beaconChainUrl, blockID)
	resp, err := b.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Beaconchain at %s: %w", url, err)
	}
	defer resp.Body.Close()
	var result blockHeaderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response for GetBlockHeader: %w", err)
	}
	return &result, nil
}

func (b *BeaconchainAdapter) getEpochFinalized(blockID string) (uint64, error) {
	header, err := b.getBlockHeader(blockID)
	if err != nil {
		return 0, fmt.Errorf("failed to get block header for blockID %s: %w", blockID, err)
	}
	slot := header.Data.Header.Message.Slot
	epoch := getEpochFromSlot(slot)
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
func (b *BeaconchainAdapter) GetValidatorLiveness(indexes []string) (map[string]bool, error) {
	epoch, err := b.getEpochFinalized("finalized")
	if err != nil {
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
	resp, err := b.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validator liveness failed: %s", resp.Status)
	}
	var result struct {
		Data []struct {
			Index  string `json:"index"`
			IsLive bool   `json:"is_live"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	liveness := make(map[string]bool)
	for _, v := range result.Data {
		liveness[v.Index] = v.IsLive
	}
	return liveness, nil
}

// GetValidatorsIndexes retrieves the validator index for each given pubkey with status active_ongoing
func (b *BeaconchainAdapter) GetValidatorsIndexes(pubkeys []string) (map[string]string, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/states/finalized/validators", b.beaconChainUrl)
	requestBody := struct {
		IDs      []string `json:"ids"`
		Statuses []string `json:"statuses"`
	}{
		IDs:      pubkeys,
		Statuses: []string{"active_ongoing"},
	}
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		return nil, err
	}
	indexMap := make(map[string]string)
	for _, v := range result.Data {
		indexMap[v.Validator.Pubkey] = v.Index
	}
	return indexMap, nil
}
