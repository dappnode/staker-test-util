package ipfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type IPFSAdapter struct {
	gateway string
	client  *http.Client
}

func NewIPFSAdapter(gateway string) *IPFSAdapter {
	return &IPFSAdapter{
		gateway: gateway,
		client:  &http.Client{},
	}
}

// GetDnpNameFromHash fetches dappnode_package.json from the given IPFS directory hash and returns the dnpName value
func (a *IPFSAdapter) GetDnpNameFromHash(ctx context.Context, ipfsHash string) (string, error) {
	url := fmt.Sprintf("%s/ipfs/%s/dappnode_package.json", a.gateway, ipfsHash)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch dappnode_package.json: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var pkg struct {
		DnpName string `json:"dnpName"`
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return "", err
	}
	if pkg.DnpName == "" {
		return "", fmt.Errorf("dnpName not found in dappnode_package.json")
	}
	return pkg.DnpName, nil
}

// GetComposeServiceName fetches docker-compose.yml from the given IPFS directory hash and returns the first and unique service name
func (a *IPFSAdapter) GetComposeServiceName(ctx context.Context, ipfsHash string) (string, error) {
	url := fmt.Sprintf("%s/ipfs/%s/docker-compose.yml", a.gateway, ipfsHash)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch docker-compose.yml: %s", resp.Status)
	}
	var compose struct {
		Services map[string]interface{} `yaml:"services"`
	}
	if err := yaml.NewDecoder(resp.Body).Decode(&compose); err != nil {
		return "", err
	}
	if len(compose.Services) != 1 {
		return "", fmt.Errorf("expected exactly one service in docker-compose.yml, got %d", len(compose.Services))
	}
	for name := range compose.Services {
		return name, nil
	}
	return "", fmt.Errorf("no service found in docker-compose.yml")
}
