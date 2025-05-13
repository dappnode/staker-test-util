package dappmanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"clients-test/internal/application/domain"
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

// Ping sends a ping request to the DappManager API
func (d *DappManagerAdapter) Ping() error {
	resp, err := d.client.Get(d.baseURL + "/ping")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	defer resp.Body.Close()

	return nil
}

// PackageInstall installs a package on the dappnode
func (d *DappManagerAdapter) PackageInstall(dnpName, versionOrIpfsHash string) error {
	url := d.baseURL + "/packageInstall"
	payload := fmt.Sprintf(`{"name": "%s", "version": "%s", "userSettings": {}, "options": {"BYPASS_CORE_RESTRICTION": true, "BYPASS_SIGNED_RESTRICTION": true}}`, dnpName, versionOrIpfsHash)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
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

// GetStakerConfig retrieves the staker configuration from the DappManager API
func (d *DappManagerAdapter) GetStakerConfig(network domain.Network) (domain.StakerConfigGetMinimal, error) {
	url := d.baseURL + "/stakerConfigGet"
	payload := fmt.Sprintf(`{"network": "%s"}`, network)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return domain.StakerConfigGetMinimal{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return domain.StakerConfigGetMinimal{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.StakerConfigGetMinimal{}, fmt.Errorf("get staker config failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.StakerConfigGetMinimal{}, err
	}

	var result domain.StakerConfigGetMinimal
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return domain.StakerConfigGetMinimal{}, err
	}
	return result, nil
}

// SetStakerConfig sets the staker configuration on the DappManager API
func (d *DappManagerAdapter) SetStakerConfig(network domain.Network, executionDnpName, consensusDnpName, mevBoostDnpName, web3signerDnpName *string, relays []string) error {
	url := d.baseURL + "/stakerConfigSet"
	requestBody := domain.StakerConfigSetRequest{
		Network:           network,
		ExecutionDnpName:  executionDnpName,
		ConsensusDnpName:  consensusDnpName,
		MevBoostDnpName:   mevBoostDnpName,
		Relays:            relays,
		Web3SignerDnpName: web3signerDnpName,
	}
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonBytes)))
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

// removePackage removes a package from the dappnode
func (d *DappManagerAdapter) removePackage(dnpName string, deleteVolumes *bool) error {
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

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonBytes)))
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

// getPackages retrieves the list of installed packages from the DappManager API
func (d *DappManagerAdapter) getPackages() ([]installedPackageMinimal, error) {
	url := d.baseURL + "/packagesGet"

	req, err := http.NewRequest("GET", url, nil)
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

// RemoveNonCorePackages removes all non-core packages from the Dappnode to clean up the system
func (d *DappManagerAdapter) RemoveNonCorePackages() error {
	packages, err := d.getPackages()
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		if !pkg.IsCore {
			deleteVolumes := true
			err := d.removePackage(pkg.DnpName, &deleteVolumes)
			if err != nil {
				return fmt.Errorf("failed to remove package %s: %w", pkg.DnpName, err)
			}
		}
	}
	return nil
}
