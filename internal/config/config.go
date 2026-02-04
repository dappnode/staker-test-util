package config

import (
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
)

var logPrefix = "Config"

// Config holds all application configuration
type Config struct {
	Mode            domain.RunMode // Required: run mode (sync or test)
	IPFSGatewayURL  string
	IPFSHash        string
	ExecutionClient string // Optional: override execution client (e.g., geth, reth, nethermind)
	ConsensusClient string // Optional: override consensus client (e.g., prysm, teku, nimbus, lodestar)
	GitHub          github.GitHubConfig
}

// ParseConfig parses CLI flags and environment variables into a Config struct
func ParseConfig() Config {
	// CLI flags
	mode := flag.String("mode", "", "Run mode: 'sync' (wait for sync only) or 'test' (full attestation test) (required, or set MODE env)")
	ipfsGatewayUrl := flag.String("ipfs-gateway-url", "", "IPFS gateway URL (required, or set IPFS_GATEWAY_URL env)")
	ipfsHash := flag.String("ipfs-hash", "", "IPFS hash for the test package (required, or set IPFS_HASH env)")

	// Optional client override flags
	executionClient := flag.String("execution-client", "", "Override execution client (geth, reth, nethermind, besu, erigon) (or set EXECUTION_CLIENT env)")
	consensusClient := flag.String("consensus-client", "", "Override consensus client (prysm, teku, nimbus, lodestar) (or set CONSENSUS_CLIENT env)")

	// GitHub flags (optional, for PR commenting)
	githubToken := flag.String("github-token", "", "GitHub token for PR commenting (or set GITHUB_TOKEN env)")
	githubRepository := flag.String("github-repository", "", "GitHub repository in format owner/repo (or set GITHUB_REPOSITORY env)")
	githubPRNumber := flag.String("github-pr-number", "", "Pull request number (or set GITHUB_PR_NUMBER env)")
	githubRunID := flag.String("github-run-id", "", "GitHub Actions run ID (or set GITHUB_RUN_ID env)")
	githubServerURL := flag.String("github-server-url", "", "GitHub server URL (or set GITHUB_SERVER_URL env)")

	flag.Parse()

	// Build config with flag values, falling back to environment variables
	// Parse and validate run mode
	modeValue := getConfigValue(*mode, "MODE", "")
	parsedMode, err := domain.ParseRunMode(modeValue)
	if err != nil {
		logger.FatalWithPrefix(logPrefix, "Invalid mode: %v. Use --mode=sync or --mode=test", err)
	}

	// Determine IPFS hash: flag/env or fallback to releases.json
	ipfsHashValue := getConfigValue(*ipfsHash, "IPFS_HASH", "")
	if ipfsHashValue == "" {
		var err error
		ipfsHashValue, err = getLatestIPFSHashFromReleases()
		if err != nil {
			logger.FatalWithPrefix(logPrefix, "Could not determine IPFS hash: %v", err)
		}
	}

	config := Config{
		Mode:            parsedMode,
		IPFSGatewayURL:  getConfigValue(*ipfsGatewayUrl, "IPFS_GATEWAY_URL", "http://ipfs.dappnode:8080"),
		IPFSHash:        ipfsHashValue,
		ExecutionClient: getConfigValue(*executionClient, "EXECUTION_CLIENT", ""),
		ConsensusClient: getConfigValue(*consensusClient, "CONSENSUS_CLIENT", ""),
		GitHub: github.ParseGitHubConfigFromEnv(
			getConfigValue(*githubToken, "GITHUB_TOKEN", ""),
			getConfigValue(*githubRepository, "GITHUB_REPOSITORY", ""),
			getGitHubPRNumber(*githubPRNumber),
			getConfigValue(*githubRunID, "GITHUB_RUN_ID", ""),
			getGitHubServerURL(*githubServerURL),
		),
	}

	return config
}

// getLatestIPFSHashFromReleases reads the latest hash from package_variants/hoodi/releases.json
func getLatestIPFSHashFromReleases() (string, error) {
	// Find the releases.json file relative to the working directory
	jsonPath := filepath.Join("package_variants", "hoodi", "releases.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", err
	}
	// Parse the JSON as a map[string]struct{hash:string}
	var releases map[string]struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(data, &releases); err != nil {
		return "", err
	}
	if len(releases) == 0 {
		return "", nil
	}
	// Take the last inserted key (latest) from the map iteration
	var latestHash string
	for _, v := range releases {
		latestHash = v.Hash
	}
	if latestHash == "" {
		return "", nil
	}
	return latestHash, nil
}

// getGitHubPRNumber gets the PR number from flag or environment variables
func getGitHubPRNumber(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if prNumber := os.Getenv("GITHUB_PR_NUMBER"); prNumber != "" {
		return prNumber
	}
	// Also check GITHUB_EVENT_NUMBER which is set in PR events
	return os.Getenv("GITHUB_EVENT_NUMBER")
}

// getGitHubServerURL gets the server URL with a default fallback
func getGitHubServerURL(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if serverURL := os.Getenv("GITHUB_SERVER_URL"); serverURL != "" {
		return serverURL
	}
	return "https://github.com"
}

// Validate checks that required configuration values are present
func (c *Config) Validate() {
	if c.IPFSHash == "" {
		logger.FatalWithPrefix(logPrefix, "IPFS hash is required. Set via --ipfs-hash flag or IPFS_HASH environment variable or build it with dappnodesdk and it will be auto-detected under releases.json.")
	}
}

// getConfigValue returns the flag value if set, otherwise falls back to the environment variable, then default
func getConfigValue(flagValue, envName, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	return defaultValue
}
