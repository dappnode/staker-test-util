package config

import (
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/logger"
	"flag"
	"os"
)

var logPrefix = "Config"

// Config holds all application configuration
type Config struct {
	IPFSGatewayURL string
	IPFSHash       string
	GitHub         github.GitHubConfig
}

// ParseConfig parses CLI flags and environment variables into a Config struct
func ParseConfig() Config {
	// CLI flags
	ipfsGatewayUrl := flag.String("ipfs-gateway-url", "", "IPFS gateway URL (required, or set IPFS_GATEWAY_URL env)")
	ipfsHash := flag.String("ipfs-hash", "", "IPFS hash for the test package (required, or set IPFS_HASH env)")

	// GitHub flags (optional, for PR commenting)
	githubToken := flag.String("github-token", "", "GitHub token for PR commenting (or set GITHUB_TOKEN env)")
	githubRepository := flag.String("github-repository", "", "GitHub repository in format owner/repo (or set GITHUB_REPOSITORY env)")
	githubPRNumber := flag.String("github-pr-number", "", "Pull request number (or set GITHUB_PR_NUMBER env)")
	githubRunID := flag.String("github-run-id", "", "GitHub Actions run ID (or set GITHUB_RUN_ID env)")
	githubServerURL := flag.String("github-server-url", "", "GitHub server URL (or set GITHUB_SERVER_URL env)")

	flag.Parse()

	// Build config with flag values, falling back to environment variables
	config := Config{
		IPFSGatewayURL: getConfigValue(*ipfsGatewayUrl, "IPFS_GATEWAY_URL", ""),
		IPFSHash:       getConfigValue(*ipfsHash, "IPFS_HASH", ""),
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
	if c.IPFSGatewayURL == "" || c.IPFSHash == "" {
		logger.FatalWithPrefix(logPrefix, "IPFS gateway URL and hash are required. Set via --ipfs-gateway-url/--ipfs-hash flags or IPFS_GATEWAY_URL/IPFS_HASH environment variables.")
	}
}
