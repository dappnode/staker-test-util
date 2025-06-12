package ipfs

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type IPFSAdapter struct {
	gateway   *string
	client    *http.Client
	logPrefix string
}

func NewIPFSAdapter(gateway *string) *IPFSAdapter {
	return &IPFSAdapter{
		gateway:   gateway,
		client:    &http.Client{},
		logPrefix: "IPFSAdapter",
	}
}

// Create unique function that returns dnpName and compose service name from IPFS hash

type IpfsResponse struct {
	DnpName     string `json:"dnpName"`
	ServiceName string `json:"serviceName"`
}

// composeFile represents the structure of docker-compose.yml with services and volumes
type composeFile struct {
	Services map[string]interface{} `yaml:"services"`
	Volumes  map[string]interface{} `yaml:"volumes"`
}

// GetDnpNameAndServiceName fetches dappnode_package.json and docker-compose.yml from the given IPFS directory hash
// and returns the dnpName and service name. Use the internal functions
func (a *IPFSAdapter) GetDnpNameAndServiceName(ctx context.Context, ipfsHash string) (domain.Pkg, error) {
	logger.DebugWithPrefix(a.logPrefix, "GetDnpNameAndServiceName: ipfsHash=%s", ipfsHash)
	dnpName, err := a.getDnpNameFromHash(ctx, ipfsHash)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "GetDnpNameAndServiceName: failed to get dnpName: %v", err)
		return domain.Pkg{}, fmt.Errorf("failed to get dnpName from IPFS hash: %w", err)
	}
	serviceName, err := a.getComposeServiceName(ctx, ipfsHash)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "GetDnpNameAndServiceName: failed to get compose service name: %v", err)
		return domain.Pkg{}, fmt.Errorf("failed to get compose service name: %w", err)
	}
	// Fetch the root volume name from compose
	volumeName, err := a.getComposeVolumeName(ctx, ipfsHash)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "GetDnpNameAndServiceName: failed to get compose volume name: %v", err)
		return domain.Pkg{}, fmt.Errorf("failed to get compose volume name: %w", err)
	}
	logger.DebugWithPrefix(a.logPrefix, "GetDnpNameAndServiceName: dnpName=%s serviceName=%s", dnpName, serviceName)
	return domain.Pkg{
		DnpName:           dnpName,
		ServiceName:       serviceName,
		ComposeVolumeName: volumeName,
		Version:           ipfsHash,
	}, nil
}

// getDnpNameFromHash fetches dappnode_package.json from the given IPFS directory hash and returns the dnpName value
func (a *IPFSAdapter) getDnpNameFromHash(ctx context.Context, ipfsHash string) (string, error) {
	url := fmt.Sprintf("%s/ipfs/%s/dappnode_package.json", *a.gateway, ipfsHash)
	logger.DebugWithPrefix(a.logPrefix, "getDnpNameFromHash: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: failed to create request: %v", err)
		return "", err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: non-200 status: %s", resp.Status)
		return "", fmt.Errorf("failed to fetch dappnode_package.json: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: failed to read body: %v", err)
		return "", err
	}
	var pkg struct {
		DnpName string `json:"name"`
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: failed to unmarshal: %v", err)
		return "", err
	}
	if pkg.DnpName == "" {
		logger.ErrorWithPrefix(a.logPrefix, "getDnpNameFromHash: dnpName not found in dappnode_package.json")
		return "", fmt.Errorf("dnpName not found in dappnode_package.json")
	}
	logger.DebugWithPrefix(a.logPrefix, "getDnpNameFromHash: dnpName=%s", pkg.DnpName)
	return pkg.DnpName, nil
}

// fetchComposeFile retrieves and decodes the docker-compose.yml into a composeFile
func (a *IPFSAdapter) fetchComposeFile(ctx context.Context, ipfsHash string) (*composeFile, error) {
	url := fmt.Sprintf("%s/ipfs/%s/docker-compose.yml", *a.gateway, ipfsHash)
	logger.DebugWithPrefix(a.logPrefix, "fetchComposeFile: url=%s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "fetchComposeFile: failed to create request: %v", err)
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "fetchComposeFile: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.ErrorWithPrefix(a.logPrefix, "fetchComposeFile: non-200 status: %s", resp.Status)
		return nil, fmt.Errorf("failed to fetch docker-compose.yml: %s", resp.Status)
	}
	var compose composeFile
	if err := yaml.NewDecoder(resp.Body).Decode(&compose); err != nil {
		logger.ErrorWithPrefix(a.logPrefix, "fetchComposeFile: failed to decode yaml: %v", err)
		return nil, err
	}
	return &compose, nil
}

// getComposeServiceName extracts the unique service name from the compose file
func (a *IPFSAdapter) getComposeServiceName(ctx context.Context, ipfsHash string) (string, error) {
	compose, err := a.fetchComposeFile(ctx, ipfsHash)
	if err != nil {
		return "", err
	}
	if len(compose.Services) != 1 {
		logger.ErrorWithPrefix(a.logPrefix, "getComposeServiceName: expected 1 service, got %d", len(compose.Services))
		return "", fmt.Errorf("expected exactly one service in docker-compose.yml, got %d", len(compose.Services))
	}
	for name := range compose.Services {
		logger.DebugWithPrefix(a.logPrefix, "getComposeServiceName: serviceName=%s", name)
		return name, nil
	}
	logger.ErrorWithPrefix(a.logPrefix, "getComposeServiceName: no service found in docker-compose.yml")
	return "", fmt.Errorf("no service found in docker-compose.yml")
}

// getComposeVolumeName extracts the unique volume name from the compose file
func (a *IPFSAdapter) getComposeVolumeName(ctx context.Context, ipfsHash string) (string, error) {
	compose, err := a.fetchComposeFile(ctx, ipfsHash)
	if err != nil {
		return "", err
	}
	if len(compose.Volumes) != 1 {
		logger.ErrorWithPrefix(a.logPrefix, "getComposeVolumeName: expected 1 volume, got %d", len(compose.Volumes))
		return "", fmt.Errorf("expected exactly one volume in docker-compose.yml, got %d", len(compose.Volumes))
	}
	for name := range compose.Volumes {
		logger.DebugWithPrefix(a.logPrefix, "getComposeVolumeName: volumeName=%s", name)
		return name, nil
	}
	logger.ErrorWithPrefix(a.logPrefix, "getComposeVolumeName: no volume found in docker-compose.yml")
	return "", fmt.Errorf("no volume found in docker-compose.yml")
}
