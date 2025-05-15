package dappmanager

import (
	"clients-test/internal/application/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DappManagerAdapter is the adapter to interact with the DappManager API
type DappManagerAdapter struct {
	baseURL string
	client  *http.Client
}

// NewDappManagerAdapter creates a new DappManagerAdapter
func NewDappManagerAdapter(baseURL string) *DappManagerAdapter {
	return &DappManagerAdapter{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Ping sends a ping request to the DappManager API with context
func (d *DappManagerAdapter) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	return nil
}

// PackageInstall installs a package on the dappnode with context
func (d *DappManagerAdapter) PackageInstall(ctx context.Context, dnpName, versionOrIpfsHash string) error {
	url := d.baseURL + "/packageInstall"
	payload := fmt.Sprintf(`{"name": "%s", "version": "%s", "userSettings": {}, "options": {"BYPASS_CORE_RESTRICTION": true, "BYPASS_SIGNED_RESTRICTION": true}}`, dnpName, versionOrIpfsHash)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("package install failed: %s", resp.Status)
	}
	return nil
}

// stakerItemMinimal represents the minimal staker item info needed
type stakerItemMinimal struct {
	DnpName    string `json:"dnpName"`
	IsSelected bool   `json:"isSelected"`
}

// StakerConfigGetMinimal represents the minimal staker config info needed
type StakerConfigGetMinimal struct {
	ExecutionClients []stakerItemMinimal `json:"executionClients"`
	ConsensusClients []stakerItemMinimal `json:"consensusClients"`
	Web3Signer       stakerItemMinimal   `json:"web3Signer"`
	MevBoost         *stakerItemMinimal  `json:"mevBoost,omitempty"`
}

// GetStakerConfig retrieves the staker configuration from the DappManager API with context
func (d *DappManagerAdapter) GetStakerConfig(ctx context.Context, network string) (StakerConfigGetMinimal, error) {
	url := d.baseURL + "/stakerConfigGet"
	payload := fmt.Sprintf(`{"network": "%s"}`, network)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload))
	if err != nil {
		return StakerConfigGetMinimal{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return StakerConfigGetMinimal{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StakerConfigGetMinimal{}, fmt.Errorf("get staker config failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return StakerConfigGetMinimal{}, err
	}

	var result StakerConfigGetMinimal
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return StakerConfigGetMinimal{}, err
	}
	return result, nil
}

// SetStakerConfig sets the staker configuration on the DappManager API with context
func (d *DappManagerAdapter) SetStakerConfig(ctx context.Context, stakerClients domain.Clients) error {
	url := d.baseURL + "/stakerConfigSet"
	jsonBytes, err := json.Marshal(stakerClients)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set staker config failed: %s", resp.Status)
	}
	return nil
}

// removePackage removes a package from the dappnode with context
func (d *DappManagerAdapter) removePackage(ctx context.Context, dnpName string, deleteVolumes *bool) error {
	url := d.baseURL + "/packageRemove"
	// Build request body
	type removeBody struct {
		DnpName       string `json:"dnpName"`
		DeleteVolumes *bool  `json:"deleteVolumes,omitempty"`
	}
	body := removeBody{
		DnpName:       dnpName,
		DeleteVolumes: deleteVolumes,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remove package failed: %s", resp.Status)
	}
	return nil
}

// installedPackageMinimal represents the minimal installed package info needed
type installedPackageMinimal struct {
	DnpName string `json:"dnpName"`
	IsCore  bool   `json:"isCore"`
}

// getPackages retrieves the list of installed packages from the DappManager API with context
func (d *DappManagerAdapter) getPackages(ctx context.Context) ([]installedPackageMinimal, error) {
	url := d.baseURL + "/packagesGet"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get packages failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var allPackages []map[string]interface{}
	err = json.Unmarshal(bodyBytes, &allPackages)
	if err != nil {
		return nil, err
	}

	var result []installedPackageMinimal
	for _, pkg := range allPackages {
		dnpName, _ := pkg["dnpName"].(string)
		isCore, _ := pkg["isCore"].(bool)
		result = append(result, installedPackageMinimal{
			DnpName: dnpName,
			IsCore:  isCore,
		})
	}
	return result, nil
}

// RemoveNonCorePackages removes all non-core packages from the Dappnode to clean up the system with context
func (d *DappManagerAdapter) RemoveNonCorePackages(ctx context.Context) error {
	packages, err := d.getPackages(ctx)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		if !pkg.IsCore {
			if strings.Contains(pkg.DnpName, "web3signer") || strings.Contains(pkg.DnpName, "mev-boost") {
				continue
			}
			deleteVolumes := true
			err := d.removePackage(ctx, pkg.DnpName, &deleteVolumes)
			if err != nil {
				return fmt.Errorf("failed to remove package %s: %w", pkg.DnpName, err)
			}
		}
	}
	return nil
}
