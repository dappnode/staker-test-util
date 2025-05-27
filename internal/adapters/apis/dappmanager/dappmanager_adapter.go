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
	baseURL   string
	client    *http.Client
	logPrefix string
}

// NewDappManagerAdapter creates a new DappManagerAdapter
func NewDappManagerAdapter() *DappManagerAdapter {
	return &DappManagerAdapter{
		baseURL:   "http://my.dappnode:7000",
		client:    &http.Client{},
		logPrefix: "DappManagerAdapter",
	}
}

// Ping sends a ping request to the DappManager API with context
func (d *DappManagerAdapter) Ping(ctx context.Context) error {
	logger.DebugWithPrefix(d.logPrefix, "Ping: url=%s", d.baseURL+"/ping")
	req, err := http.NewRequestWithContext(ctx, "GET", d.baseURL+"/ping", nil)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "Ping: failed to create request: %v", err)
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "Ping: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "Ping: non-200 status: %s", resp.Status)
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	logger.DebugWithPrefix(d.logPrefix, "Ping: success")
	return nil
}

// PackageInstall installs a package on the dappnode with context
func (d *DappManagerAdapter) PackageInstall(ctx context.Context, pkg domain.Pkg) error {
	url := d.baseURL + "/packageInstall"
	payload := fmt.Sprintf(`{"name": "%s", "version": "%s", "userSettings": {}, "options": {"BYPASS_CORE_RESTRICTION": true, "BYPASS_SIGNED_RESTRICTION": true}}`, pkg.DnpName, pkg.Version)
	logger.DebugWithPrefix(d.logPrefix, "PackageInstall: url=%s payload=%s", url, payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "PackageInstall: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "PackageInstall: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "PackageInstall: non-200 status: %s", resp.Status)
		return fmt.Errorf("package install failed: %s", resp.Status)
	}
	logger.DebugWithPrefix(d.logPrefix, "PackageInstall: success for pkg=%+v", pkg)
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
	logger.DebugWithPrefix(d.logPrefix, "GetStakerConfig: url=%s payload=%s", url, payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "GetStakerConfig: failed to create request: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "GetStakerConfig: request failed: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "GetStakerConfig: non-200 status: %s", resp.Status)
		return StakerConfigGetMinimal{}, fmt.Errorf("get staker config failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "GetStakerConfig: failed to read response body: %v", err)
		return StakerConfigGetMinimal{}, err
	}

	var result StakerConfigGetMinimal
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "GetStakerConfig: failed to unmarshal response: %v", err)
		return StakerConfigGetMinimal{}, err
	}
	logger.DebugWithPrefix(d.logPrefix, "GetStakerConfig: result=%+v", result)
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
			"web3signerDnpName": stakerClients.Web3SignerDnpName,
			"relays":            stakerClients.Relays,
		},
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "SetStakerConfig: failed to marshal payload: %v", err)
		return err
	}
	logger.DebugWithPrefix(d.logPrefix, "SetStakerConfig: url=%s payload=%s", url, string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "SetStakerConfig: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "SetStakerConfig: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "SetStakerConfig: non-200 status: %s", resp.Status)
		return fmt.Errorf("set staker config failed: %s", resp.Status)
	}
	logger.DebugWithPrefix(d.logPrefix, "SetStakerConfig: success for stakerClients=%+v", stakerClients)
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
		logger.ErrorWithPrefix(d.logPrefix, "removePackage: failed to marshal body: %v", err)
		return err
	}
	logger.DebugWithPrefix(d.logPrefix, "removePackage: url=%s body=%s", url, string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "removePackage: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "removePackage: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "removePackage: non-200 status: %s", resp.Status)
		return fmt.Errorf("remove package failed: %s", resp.Status)
	}
	logger.DebugWithPrefix(d.logPrefix, "removePackage: success for dnpName=%s", dnpName)
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
	logger.DebugWithPrefix(d.logPrefix, "getPackages: url=%s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "getPackages: failed to create request: %v", err)
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "getPackages: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(d.logPrefix, "getPackages: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("get packages failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "getPackages: failed to read response body: %v", err)
		return nil, err
	}

	var allPackages []map[string]interface{}
	err = json.Unmarshal(bodyBytes, &allPackages)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "getPackages: failed to unmarshal response: %v", err)
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
	logger.DebugWithPrefix(d.logPrefix, "getPackages: result=%+v", result)
	return result, nil
}

// RemoveNonCorePackages removes all non-core packages from the Dappnode to clean up the system with context
func (d *DappManagerAdapter) RemoveNonCorePackages(ctx context.Context) error {
	logger.DebugWithPrefix(d.logPrefix, "RemoveNonCorePackages: called")
	packages, err := d.getPackages(ctx)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "RemoveNonCorePackages: failed to get packages: %v", err)
		return err
	}
	for _, pkg := range packages {
		if !pkg.IsCore {
			deleteVolumes := true
			// Do not remove web3signer volumes
			if strings.Contains(pkg.DnpName, "web3signer") {
				deleteVolumes = false
			}

			logger.DebugWithPrefix(d.logPrefix, "RemoveNonCorePackages: removing dnpName=%s deleteVolumes=%v", pkg.DnpName, deleteVolumes)
			err := d.removePackage(ctx, pkg.DnpName, &deleteVolumes)
			if err != nil {
				logger.ErrorWithPrefix(d.logPrefix, "RemoveNonCorePackages: failed to remove package %s: %v", pkg.DnpName, err)
				return fmt.Errorf("failed to remove package %s: %w", pkg.DnpName, err)
			}
		}
	}
	logger.DebugWithPrefix(d.logPrefix, "RemoveNonCorePackages: success")
	return nil
}
