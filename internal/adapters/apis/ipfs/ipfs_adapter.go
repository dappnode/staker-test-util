package ipfs

import (
	"clients-test/internal/application/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type IPFSAdapter struct {
	gateway *string
	client  *http.Client
}

func NewIPFSAdapter(gateway *string) *IPFSAdapter {
	return &IPFSAdapter{
		gateway: gateway,
		client:  &http.Client{},
	}
}

// Create unique function that returns dnpName and compose service name from IPFS hash

type IpfsResponse struct {
	DnpName     string `json:"dnpName"`
	ServiceName string `json:"serviceName"`
}

// GetDnpNameAndServiceName fetches dappnode_package.json and docker-compose.yml from the given IPFS directory hash
// and returns the dnpName and service name. Use the internal functions
func (a *IPFSAdapter) GetDnpNameAndServiceName(ctx context.Context, ipfsHash string) (domain.Pkg, error) {
	dnpName, err := a.getDnpNameFromHash(ctx, ipfsHash)
	if err != nil {
		return domain.Pkg{}, fmt.Errorf("failed to get dnpName from IPFS hash: %w", err)
	}
	serviceName, err := a.getComposeServiceName(ctx, ipfsHash)
	if err != nil {
		return domain.Pkg{}, fmt.Errorf("failed to get compose service name: %w", err)
	}
	return domain.Pkg{
		DnpName:     dnpName,
		ServiceName: serviceName,
		Version:     ipfsHash,
	}, nil
}

// getDnpNameFromHash fetches dappnode_package.json from the given IPFS directory hash and returns the dnpName value
func (a *IPFSAdapter) getDnpNameFromHash(ctx context.Context, ipfsHash string) (string, error) {
	url := fmt.Sprintf("%s/ipfs/%s/dappnode_package.json", *a.gateway, ipfsHash)
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

// getComposeServiceName fetches docker-compose.yml from the given IPFS directory hash and returns the first and unique service name
func (a *IPFSAdapter) getComposeServiceName(ctx context.Context, ipfsHash string) (string, error) {
	url := fmt.Sprintf("%s/ipfs/%s/docker-compose.yml", *a.gateway, ipfsHash)
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
