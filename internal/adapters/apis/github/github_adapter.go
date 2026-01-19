package github

import (
	"bytes"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// GitHubConfig holds the configuration for GitHub API interactions
type GitHubConfig struct {
	Token      string // GitHub token (PAT or GITHUB_TOKEN from Actions)
	Owner      string // Repository owner (e.g., "dappnode")
	Repo       string // Repository name (e.g., "staker-test-util")
	PRNumber   int    // Pull request number (0 if not in PR context)
	RunID      string // GitHub Actions run ID for linking to CI logs
	RunURL     string // Full URL to the GitHub Actions run
	ServerURL  string // GitHub server URL (for GitHub Enterprise)
	Repository string // Full repository name (owner/repo)
}

// GitHubAdapter handles interactions with the GitHub API
type GitHubAdapter struct {
	config    GitHubConfig
	client    *http.Client
	logPrefix string
	baseURL   string
}

// NewGitHubAdapter creates a new GitHub adapter
func NewGitHubAdapter(config GitHubConfig) *GitHubAdapter {
	baseURL := "https://api.github.com"
	if config.ServerURL != "" && config.ServerURL != "https://github.com" {
		// For GitHub Enterprise
		baseURL = strings.TrimSuffix(config.ServerURL, "/") + "/api/v3"
	}

	return &GitHubAdapter{
		config:    config,
		client:    &http.Client{},
		logPrefix: "GitHubAdapter",
		baseURL:   baseURL,
	}
}

// IsEnabled returns true if the GitHub adapter is properly configured
func (g *GitHubAdapter) IsEnabled() bool {
	return g.config.Token != "" && g.config.Owner != "" && g.config.Repo != "" && g.config.PRNumber > 0
}

// IssueComment represents a GitHub issue/PR comment
type IssueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// CommentOnPR creates or updates a comment on a pull request with the test report
func (g *GitHubAdapter) CommentOnPR(ctx context.Context, report *domain.TestReport) error {
	if !g.IsEnabled() {
		logger.InfoWithPrefix(g.logPrefix, "GitHub integration not enabled (missing token, repo info, or PR number), skipping PR comment")
		return nil
	}

	// Generate markdown report
	markdown := report.ToMarkdown()

	// Add CI logs link if available
	if g.config.RunURL != "" {
		markdown += fmt.Sprintf("\n---\n📋 [View full CI logs](%s)\n", g.config.RunURL)
	}

	// Add a signature to identify our comments for updates
	signature := "\n\n<!-- staker-test-report -->"
	markdown += signature

	// Check for existing comment to update
	existingCommentID, err := g.findExistingComment(ctx, signature)
	if err != nil {
		logger.WarnWithPrefix(g.logPrefix, "Failed to find existing comment, will create new: %v", err)
	}

	if existingCommentID > 0 {
		// Update existing comment
		err = g.updateComment(ctx, existingCommentID, markdown)
		if err != nil {
			return fmt.Errorf("failed to update PR comment: %w", err)
		}
		logger.InfoWithPrefix(g.logPrefix, "Updated existing comment on PR #%d", g.config.PRNumber)
	} else {
		// Create new comment
		err = g.createComment(ctx, markdown)
		if err != nil {
			return fmt.Errorf("failed to create PR comment: %w", err)
		}
		logger.InfoWithPrefix(g.logPrefix, "Created new comment on PR #%d", g.config.PRNumber)
	}

	return nil
}

// findExistingComment looks for an existing comment with our signature
func (g *GitHubAdapter) findExistingComment(ctx context.Context, signature string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments",
		g.baseURL, g.config.Owner, g.config.Repo, g.config.PRNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to list comments: %s - %s", resp.Status, string(body))
	}

	var comments []IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return 0, err
	}

	for _, comment := range comments {
		if strings.Contains(comment.Body, signature) {
			return comment.ID, nil
		}
	}

	return 0, nil
}

// createComment creates a new comment on the PR
func (g *GitHubAdapter) createComment(ctx context.Context, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments",
		g.baseURL, g.config.Owner, g.config.Repo, g.config.PRNumber)

	payload := map[string]string{"body": body}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	g.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create comment: %s - %s", resp.Status, string(body))
	}

	return nil
}

// updateComment updates an existing comment
func (g *GitHubAdapter) updateComment(ctx context.Context, commentID int64, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d",
		g.baseURL, g.config.Owner, g.config.Repo, commentID)

	payload := map[string]string{"body": body}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	g.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update comment: %s - %s", resp.Status, string(body))
	}

	return nil
}

// setHeaders sets common headers for GitHub API requests
func (g *GitHubAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+g.config.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// ParseGitHubConfigFromEnv creates a GitHubConfig from environment variables
// This is typically used in GitHub Actions context
func ParseGitHubConfigFromEnv(token, repository, prNumber, runID, serverURL string) GitHubConfig {
	config := GitHubConfig{
		Token:      token,
		Repository: repository,
		RunID:      runID,
		ServerURL:  serverURL,
	}

	// Parse repository into owner/repo
	if repository != "" {
		parts := strings.SplitN(repository, "/", 2)
		if len(parts) == 2 {
			config.Owner = parts[0]
			config.Repo = parts[1]
		}
	}

	// Parse PR number
	if prNumber != "" {
		if num, err := strconv.Atoi(prNumber); err == nil {
			config.PRNumber = num
		}
	}

	// Build run URL
	if serverURL != "" && repository != "" && runID != "" {
		config.RunURL = fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repository, runID)
	}

	return config
}
