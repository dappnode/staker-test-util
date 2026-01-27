package dappmanager

import (
	"bytes"
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
func NewDappManagerAdapter() *DappManagerAdapter {
	return &DappManagerAdapter{
		baseURL: "http://my.dappnode:7000",
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
func (d *DappManagerAdapter) PackageInstall(ctx context.Context, pkg domain.Pkg) error {
	url := d.baseURL + "/packageInstall"
	payload := fmt.Sprintf(`{"name": "%s", "version": "%s", "userSettings": {}, "options": {"BYPASS_CORE_RESTRICTION": true, "BYPASS_SIGNED_RESTRICTION": true}}`, pkg.DnpName, pkg.Version)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
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
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
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
// Returns a list of packages that were skipped and any errors encountered
func (d *DappManagerAdapter) RemoveNonCorePackages(ctx context.Context) (skipped []string, errors []error) {
	packages, err := d.getPackages(ctx)
	if err != nil {
		return nil, []error{err}
	}
	for _, pkg := range packages {
		if !pkg.IsCore {
			// skip if pkg.DnpName includes web3signer or mev-boost
			if strings.Contains(pkg.DnpName, "web3signer") || strings.Contains(pkg.DnpName, "mev-boost") || strings.Contains(pkg.DnpName, "dms") {
				skipped = append(skipped, pkg.DnpName)
				continue
			}
			var deleteVolumes bool
			if isExecutionPackage(pkg.DnpName) {
				deleteVolumes = false
			} else {
				deleteVolumes = true
			}
			err := d.removePackage(ctx, pkg.DnpName, &deleteVolumes)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to remove package %s: %w", pkg.DnpName, err))
				continue
			}
		}
	}
	return skipped, errors
}

// isExecutionPackage returns true if the dnpName is an execution client
func isExecutionPackage(dnpName string) bool {
	execs := []string{"geth", "besu", "erigon", "reth", "nethermind"}
	lower := strings.ToLower(dnpName)
	for _, exec := range execs {
		if strings.Contains(lower, exec) {
			return true
		}
	}
	return false
}
