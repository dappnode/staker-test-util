package domain

import (
	"fmt"
	"strings"
	"time"
)

// TimingEntry represents a single timed operation
type TimingEntry struct {
	Operation string        `json:"operation"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

// ContainerErrorLog represents error logs from a container
type ContainerErrorLog struct {
	ContainerName string   `json:"containerName"`
	ErrorLines    []string `json:"errorLines"`
}

// TestReport holds all information for the test report
type TestReport struct {
	// Client configuration
	ExecutionDnpName  string `json:"executionDnpName"`
	ConsensusDnpName  string `json:"consensusDnpName"`
	Web3SignerDnpName string `json:"web3signerDnpName"`
	MevBoostDnpName   string `json:"mevBoostDnpName"`
	Network           string `json:"network"`

	// Client versions (before/after install)
	ExecutionClientVersionBefore string `json:"executionClientVersionBefore,omitempty"`
	ExecutionClientVersionAfter  string `json:"executionClientVersionAfter,omitempty"`
	ConsensusClientVersionBefore string `json:"consensusClientVersionBefore,omitempty"`
	ConsensusClientVersionAfter  string `json:"consensusClientVersionAfter,omitempty"`
	SnapshotClientVersion        string `json:"snapshotClientVersion,omitempty"`

	// blocknumber of the snapshot used
	SnapshotBlockNumber uint64 `json:"snapshotBlockNumber,omitempty"`

	// What type of client is being tested
	TestedClientType string `json:"testedClientType,omitempty"` // "execution" or "consensus"

	// Test execution info
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`

	// Timing measurements
	EnsureTimings  []TimingEntry `json:"ensureTimings"`
	ExecuteTimings []TimingEntry `json:"executeTimings"`

	// Container error logs
	ContainerErrors []ContainerErrorLog `json:"containerErrors"`

	// Final result
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// NewTestReport creates a new TestReport from StakerConfig
func NewTestReport(config StakerConfig) *TestReport {
	return &TestReport{
		ExecutionDnpName:  config.ExecutionDnpName,
		ConsensusDnpName:  config.ConsensusDnpName,
		Web3SignerDnpName: config.Web3SignerDnpName,
		MevBoostDnpName:   config.MevBoostDnpName,
		Network:           config.Network,
		StartTime:         time.Now(),
		EnsureTimings:     make([]TimingEntry, 0),
		ExecuteTimings:    make([]TimingEntry, 0),
		ContainerErrors:   make([]ContainerErrorLog, 0),
		Success:           true,
	}
}

// AddEnsureTiming adds a timing entry for an ensure operation
func (r *TestReport) AddEnsureTiming(operation string, duration time.Duration, success bool, err error) {
	entry := TimingEntry{
		Operation: operation,
		Duration:  duration,
		Success:   success,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	r.EnsureTimings = append(r.EnsureTimings, entry)
}

// AddExecuteTiming adds a timing entry for an execute operation
func (r *TestReport) AddExecuteTiming(operation string, duration time.Duration, success bool, err error) {
	entry := TimingEntry{
		Operation: operation,
		Duration:  duration,
		Success:   success,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	r.ExecuteTimings = append(r.ExecuteTimings, entry)
}

// AddContainerErrors adds error logs for a container
func (r *TestReport) AddContainerErrors(containerName string, errorLines []string) {
	if len(errorLines) > 0 {
		r.ContainerErrors = append(r.ContainerErrors, ContainerErrorLog{
			ContainerName: containerName,
			ErrorLines:    errorLines,
		})
	}
}

// SetResult sets the final test result
func (r *TestReport) SetResult(success bool, err error) {
	r.EndTime = time.Now()
	r.Success = success
	if err != nil {
		r.ErrorMessage = err.Error()
	}
}

// TotalDuration returns the total test duration
func (r *TestReport) TotalDuration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// ToMarkdown generates a markdown report for GitHub PR comment
func (r *TestReport) ToMarkdown() string {
	var sb strings.Builder

	// Header with result emoji
	if r.Success {
		sb.WriteString("## ✅ Staker Test Report - PASSED\n\n")
	} else {
		sb.WriteString("## ❌ Staker Test Report - FAILED\n\n")
	}

	// Clients Used section
	sb.WriteString("### 📦 Clients Used\n\n")
	sb.WriteString("| Component | DNP Name |\n")
	sb.WriteString("|-----------|----------|\n")
	sb.WriteString(fmt.Sprintf("| Execution | `%s` |\n", r.ExecutionDnpName))
	sb.WriteString(fmt.Sprintf("| Consensus | `%s` |\n", r.ConsensusDnpName))
	sb.WriteString(fmt.Sprintf("| Web3Signer | `%s` |\n", r.Web3SignerDnpName))
	sb.WriteString(fmt.Sprintf("| MEV Boost | `%s` |\n", r.MevBoostDnpName))
	sb.WriteString(fmt.Sprintf("| Network | `%s` |\n", r.Network))
	sb.WriteString("\n")

	// Version tracking section
	if r.TestedClientType != "" {
		sb.WriteString("### 🔖 Version Tracking\n\n")
		if r.TestedClientType == "execution" {
			sb.WriteString("**Tested Client:** Execution\n\n")
			sb.WriteString("| Stage | Version |\n")
			sb.WriteString("|-------|---------|\n")
			if r.ExecutionClientVersionBefore != "" {
				sb.WriteString(fmt.Sprintf("| Before Install | `%s` |\n", r.ExecutionClientVersionBefore))
			} else {
				sb.WriteString("| Before Install | _not available_ |\n")
			}
			if r.ExecutionClientVersionAfter != "" {
				sb.WriteString(fmt.Sprintf("| After Install | `%s` |\n", r.ExecutionClientVersionAfter))
			} else {
				sb.WriteString("| After Install | _not available_ |\n")
			}
			if r.SnapshotClientVersion != "" {
				sb.WriteString(fmt.Sprintf("| Snapshot | `%s` |\n", r.SnapshotClientVersion))
			}
			if r.SnapshotBlockNumber > 0 {
				sb.WriteString(fmt.Sprintf("| Snapshot Block | `%d` |\n", r.SnapshotBlockNumber))
			}
		} else if r.TestedClientType == "consensus" {
			sb.WriteString("**Tested Client:** Consensus\n\n")
			sb.WriteString("| Stage | Version |\n")
			sb.WriteString("|-------|---------|\n")
			if r.ConsensusClientVersionBefore != "" {
				sb.WriteString(fmt.Sprintf("| Before Install | `%s` |\n", r.ConsensusClientVersionBefore))
			} else {
				sb.WriteString("| Before Install | _not available_ |\n")
			}
			if r.ConsensusClientVersionAfter != "" {
				sb.WriteString(fmt.Sprintf("| After Install | `%s` |\n", r.ConsensusClientVersionAfter))
			} else {
				sb.WriteString("| After Install | _not available_ |\n")
			}
		}
		sb.WriteString("\n")
	}

	// Timing section
	sb.WriteString("### ⏱️ Timing Measurements\n\n")

	if len(r.EnsureTimings) > 0 {
		sb.WriteString("#### Environment Setup\n\n")
		sb.WriteString("| Operation | Duration | Status |\n")
		sb.WriteString("|-----------|----------|--------|\n")
		for _, t := range r.EnsureTimings {
			status := "✅"
			if !t.Success {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", t.Operation, t.Duration.Round(time.Millisecond), status))
		}
		sb.WriteString("\n")
	}

	if len(r.ExecuteTimings) > 0 {
		sb.WriteString("#### Test Execution\n\n")
		sb.WriteString("| Operation | Duration | Status |\n")
		sb.WriteString("|-----------|----------|--------|\n")
		for _, t := range r.ExecuteTimings {
			status := "✅"
			if !t.Success {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", t.Operation, t.Duration.Round(time.Millisecond), status))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**Total Duration:** %s\n\n", r.TotalDuration().Round(time.Second)))

	// Error logs section
	if len(r.ContainerErrors) > 0 {
		sb.WriteString("### 🔴 Container Error Logs\n\n")
		sb.WriteString("> ⚠️ Showing up to 3 error lines per container. See CI logs for complete details.\n\n")
		for _, ce := range r.ContainerErrors {
			sb.WriteString(fmt.Sprintf("**%s:**\n", ce.ContainerName))
			sb.WriteString("```\n")
			for _, line := range ce.ErrorLines {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("```\n\n")
		}
	}

	// Error message if failed
	if !r.Success && r.ErrorMessage != "" {
		sb.WriteString("### ❌ Error Details\n\n")
		sb.WriteString("```\n")
		sb.WriteString(r.ErrorMessage + "\n")
		sb.WriteString("```\n")
	}

	return sb.String()
}

// ToConsoleString generates a console-friendly report
func (r *TestReport) ToConsoleString() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("========================================\n")
	if r.Success {
		sb.WriteString("       STAKER TEST REPORT - PASSED      \n")
	} else {
		sb.WriteString("       STAKER TEST REPORT - FAILED      \n")
	}
	sb.WriteString("========================================\n\n")

	// Clients Used
	sb.WriteString("CLIENTS USED:\n")
	sb.WriteString(fmt.Sprintf("  Execution:   %s\n", r.ExecutionDnpName))
	sb.WriteString(fmt.Sprintf("  Consensus:   %s\n", r.ConsensusDnpName))
	sb.WriteString(fmt.Sprintf("  Web3Signer:  %s\n", r.Web3SignerDnpName))
	sb.WriteString(fmt.Sprintf("  MEV Boost:   %s\n", r.MevBoostDnpName))
	sb.WriteString(fmt.Sprintf("  Network:     %s\n", r.Network))
	sb.WriteString("\n")

	// Version tracking
	if r.TestedClientType != "" {
		sb.WriteString("VERSION TRACKING:\n")
		sb.WriteString(fmt.Sprintf("  Tested Client Type: %s\n", r.TestedClientType))
		if r.TestedClientType == "execution" {
			if r.ExecutionClientVersionBefore != "" {
				sb.WriteString(fmt.Sprintf("  Before Install:     %s\n", r.ExecutionClientVersionBefore))
			}
			if r.ExecutionClientVersionAfter != "" {
				sb.WriteString(fmt.Sprintf("  After Install:      %s\n", r.ExecutionClientVersionAfter))
			}
			if r.SnapshotClientVersion != "" {
				sb.WriteString(fmt.Sprintf("  Snapshot Version:   %s\n", r.SnapshotClientVersion))
			}
			if r.SnapshotBlockNumber > 0 {
				sb.WriteString(fmt.Sprintf("  Snapshot Block:     %d\n", r.SnapshotBlockNumber))
			}
		} else if r.TestedClientType == "consensus" {
			if r.ConsensusClientVersionBefore != "" {
				sb.WriteString(fmt.Sprintf("  Before Install:     %s\n", r.ConsensusClientVersionBefore))
			}
			if r.ConsensusClientVersionAfter != "" {
				sb.WriteString(fmt.Sprintf("  After Install:      %s\n", r.ConsensusClientVersionAfter))
			}
		}
		sb.WriteString("\n")
	}

	// Timing
	sb.WriteString("TIMING MEASUREMENTS:\n")
	if len(r.EnsureTimings) > 0 {
		sb.WriteString("  Environment Setup:\n")
		for _, t := range r.EnsureTimings {
			status := "OK"
			if !t.Success {
				status = "FAILED"
			}
			sb.WriteString(fmt.Sprintf("    - %-30s %12s [%s]\n", t.Operation, t.Duration.Round(time.Millisecond), status))
		}
	}
	if len(r.ExecuteTimings) > 0 {
		sb.WriteString("  Test Execution:\n")
		for _, t := range r.ExecuteTimings {
			status := "OK"
			if !t.Success {
				status = "FAILED"
			}
			sb.WriteString(fmt.Sprintf("    - %-30s %12s [%s]\n", t.Operation, t.Duration.Round(time.Millisecond), status))
		}
	}
	sb.WriteString(fmt.Sprintf("\n  Total Duration: %s\n", r.TotalDuration().Round(time.Second)))
	sb.WriteString("\n")

	// Error logs
	if len(r.ContainerErrors) > 0 {
		sb.WriteString("CONTAINER ERROR LOGS:\n")
		sb.WriteString("  (See CI logs for complete details)\n")
		for _, ce := range r.ContainerErrors {
			sb.WriteString(fmt.Sprintf("  [%s]:\n", ce.ContainerName))
			for _, line := range ce.ErrorLines {
				sb.WriteString(fmt.Sprintf("    %s\n", line))
			}
		}
		sb.WriteString("\n")
	}

	// Error message
	if !r.Success && r.ErrorMessage != "" {
		sb.WriteString("ERROR DETAILS:\n")
		sb.WriteString(fmt.Sprintf("  %s\n", r.ErrorMessage))
	}

	sb.WriteString("========================================\n")

	return sb.String()
}
