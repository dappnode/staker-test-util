package dappmanager

import (
	"bytes"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
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
func NewDappManagerAdapter() *DappManagerAdapter {
	return &DappManagerAdapter{
		baseURL: "http://my.dappnode:7000",
		client:  &http.Client{},
	}
}

// Ping sends a ping request to the DappManager API with context
func (d *DappManagerAdapter) Ping(ctx context.Context) error {
	logger.Debug("[DappManagerAdapter] Ping: url=%s", d.baseURL+"/ping")
	req, err := http.NewRequestWithContext(ctx, "GET", d.baseURL+"/ping", nil)
	if err != nil {
		logger.Error("[DappManagerAdapter] Ping: failed to create request: %v", err)
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] Ping: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] Ping: non-200 status: %s", resp.Status)
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	logger.Debug("[DappManagerAdapter] Ping: success")
	return nil
}

// PackageInstall installs a package on the dappnode with context
func (d *DappManagerAdapter) PackageInstall(ctx context.Context, pkg domain.Pkg) error {
	url := d.baseURL + "/packageInstall"
	payload := fmt.Sprintf(`{"name": "%s", "version": "%s", "userSettings": {}, "options": {"BYPASS_CORE_RESTRICTION": true, "BYPASS_SIGNED_RESTRICTION": true}}`, pkg.DnpName, pkg.Version)
	logger.Debug("[DappManagerAdapter] PackageInstall: url=%s payload=%s", url, payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
	if err != nil {
		logger.Error("[DappManagerAdapter] PackageInstall: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] PackageInstall: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] PackageInstall: non-200 status: %s", resp.Status)
		return fmt.Errorf("package install failed: %s", resp.Status)
	}
	logger.Debug("[DappManagerAdapter] PackageInstall: success for pkg=%+v", pkg)
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
	logger.Debug("[DappManagerAdapter] GetStakerConfig: url=%s payload=%s", url, payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
	if err != nil {
		logger.Error("[DappManagerAdapter] GetStakerConfig: failed to create request: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] GetStakerConfig: request failed: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] GetStakerConfig: non-200 status: %s", resp.Status)
		return StakerConfigGetMinimal{}, fmt.Errorf("get staker config failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[DappManagerAdapter] GetStakerConfig: failed to read response body: %v", err)
		return StakerConfigGetMinimal{}, err
	}

	var result StakerConfigGetMinimal
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		logger.Error("[DappManagerAdapter] GetStakerConfig: failed to unmarshal response: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	logger.Debug("[DappManagerAdapter] GetStakerConfig: result=%+v", result)
	return result, nil
}

// SetStakerConfig sets the staker configuration on the DappManager API with context
func (d *DappManagerAdapter) SetStakerConfig(ctx context.Context, stakerClients domain.StakerConfig) error {
	url := d.baseURL + "/stakerConfigSet"
	payload := map[string]interface{}{
		"stakerConfig": map[string]interface{}{
			"network":           stakerClients.Network,
			"executionDnpName":  stakerClients.ExecutionDnpName,
			"consensusDnpName":  stakerClients.ConsensusDnpName,
			"mevBoostDnpName":   stakerClients.MevBoostDnpName,
			"web3SignerDnpName": stakerClients.Web3SignerDnpName,
			"relays":            stakerClients.Relays,
		},
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error("[DappManagerAdapter] SetStakerConfig: failed to marshal payload: %v", err)
		return err
	}
	logger.Debug("[DappManagerAdapter] SetStakerConfig: url=%s payload=%s", url, string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.Error("[DappManagerAdapter] SetStakerConfig: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] SetStakerConfig: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] SetStakerConfig: non-200 status: %s", resp.Status)
		return fmt.Errorf("set staker config failed: %s", resp.Status)
	}
	logger.Debug("[DappManagerAdapter] SetStakerConfig: success for stakerClients=%+v", stakerClients)
	return nil
}

// removePackage removes a package from the dappnode with context
func (d *DappManagerAdapter) removePackage(ctx context.Context, dnpName string, deleteVolumes *bool) error {
	url := d.baseURL + "/packageRemove"
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
		logger.Error("[DappManagerAdapter] removePackage: failed to marshal body: %v", err)
		return err
	}
	logger.Debug("[DappManagerAdapter] removePackage: url=%s body=%s", url, string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.Error("[DappManagerAdapter] removePackage: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] removePackage: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] removePackage: non-200 status: %s", resp.Status)
		return fmt.Errorf("remove package failed: %s", resp.Status)
	}
	logger.Debug("[DappManagerAdapter] removePackage: success for dnpName=%s", dnpName)
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
	logger.Debug("[DappManagerAdapter] getPackages: url=%s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("[DappManagerAdapter] getPackages: failed to create request: %v", err)
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		logger.Error("[DappManagerAdapter] getPackages: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("[DappManagerAdapter] getPackages: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("get packages failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[DappManagerAdapter] getPackages: failed to read response body: %v", err)
		return nil, err
	}

	var allPackages []map[string]interface{}
	err = json.Unmarshal(bodyBytes, &allPackages)
	if err != nil {
		logger.Error("[DappManagerAdapter] getPackages: failed to unmarshal response: %v", err)
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
	logger.Debug("[DappManagerAdapter] getPackages: result=%+v", result)
	return result, nil
}

// RemoveNonCorePackages removes all non-core packages from the Dappnode to clean up the system with context
func (d *DappManagerAdapter) RemoveNonCorePackages(ctx context.Context) error {
	logger.Debug("[DappManagerAdapter] RemoveNonCorePackages: called")
	packages, err := d.getPackages(ctx)
	if err != nil {
		logger.Error("[DappManagerAdapter] RemoveNonCorePackages: failed to get packages: %v", err)
		return err
	}
	for _, pkg := range packages {
		if !pkg.IsCore {
			deleteVolumes := true
			// Do not remove web3signer volumes
			if strings.Contains(pkg.DnpName, "web3signer") {
				deleteVolumes = false
			}

			logger.Debug("[DappManagerAdapter] RemoveNonCorePackages: removing dnpName=%s deleteVolumes=%v", pkg.DnpName, deleteVolumes)
			err := d.removePackage(ctx, pkg.DnpName, &deleteVolumes)
			if err != nil {
				logger.Error("[DappManagerAdapter] RemoveNonCorePackages: failed to remove package %s: %v", pkg.DnpName, err)
				return fmt.Errorf("failed to remove package %s: %w", pkg.DnpName, err)
			}
		}
	}
	logger.Debug("[DappManagerAdapter] RemoveNonCorePackages: success")
	return nil
}
