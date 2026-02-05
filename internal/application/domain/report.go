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
	// Mode determines the report title (sync vs test)
	Mode RunMode `json:"mode,omitempty"`

	// Client configuration
	ExecutionDnpName     string `json:"executionDnpName"`
	ExecutionDnpVersion  string `json:"executionDnpVersion,omitempty"`
	ConsensusDnpName     string `json:"consensusDnpName"`
	ConsensusDnpVersion  string `json:"consensusDnpVersion,omitempty"`
	Web3SignerDnpName    string `json:"web3signerDnpName"`
	Web3SignerDnpVersion string `json:"web3signerDnpVersion,omitempty"`
	MevBoostDnpName      string `json:"mevBoostDnpName"`
	MevBoostDnpVersion   string `json:"mevBoostDnpVersion,omitempty"`
	Network              string `json:"network"`

	// Client versions (before/after install)
	ExecutionClientVersionBefore string `json:"executionClientVersionBefore,omitempty"`
	ExecutionClientVersionAfter  string `json:"executionClientVersionAfter,omitempty"`
	ConsensusClientVersionBefore string `json:"consensusClientVersionBefore,omitempty"`
	ConsensusClientVersionAfter  string `json:"consensusClientVersionAfter,omitempty"`

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

	// Beaconcha.in URLs
	BeaconchainEpochURL      string   `json:"beaconchainEpochURL,omitempty"`
	BeaconchainValidatorURLs []string `json:"beaconchainValidatorURLs,omitempty"`
}

func (r *TestReport) reportTitle() string {
	if r.Mode.IsSync() {
		return "SYNC TEST REPORT"
	}
	if r.Mode.IsTest() {
		return "PROOF OF ATTESTATION TEST REPORT"
	}
	return "TEST REPORT"
}

// NewTestReport creates a new TestReport from mode + StakerConfig
func NewTestReport(mode RunMode, config StakerConfig) *TestReport {
	return &TestReport{
		Mode:              mode,
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
		sb.WriteString(fmt.Sprintf("## ✅ %s - PASSED\n\n", r.reportTitle()))
	} else {
		sb.WriteString(fmt.Sprintf("## ❌ %s - FAILED\n\n", r.reportTitle()))
	}

	// Attestation section (if URLs present)
	if r.BeaconchainEpochURL != "" || len(r.BeaconchainValidatorURLs) > 0 {
		sb.WriteString("### 📝 Attestation\n\n")
		if r.Success && r.BeaconchainEpochURL != "" {
			sb.WriteString(fmt.Sprintf("- [Epoch on beaconcha.in](%s)\n", r.BeaconchainEpochURL))
		}
		for _, vurl := range r.BeaconchainValidatorURLs {
			sb.WriteString(fmt.Sprintf("- [Validator on beaconcha.in](%s)\n", vurl))
		}
		sb.WriteString("\n")
	}

	// Clients Used section
	sb.WriteString("### 📦 Clients Used\n\n")
	sb.WriteString("| Component | DNP Name | Version |\n")
	sb.WriteString("|-----------|----------|---------|\n")
	sb.WriteString(fmt.Sprintf("| Execution | `%s` | `%s` |\n", r.ExecutionDnpName, r.ExecutionDnpVersion))
	sb.WriteString(fmt.Sprintf("| Consensus | `%s` | `%s` |\n", r.ConsensusDnpName, r.ConsensusDnpVersion))
	sb.WriteString(fmt.Sprintf("| Web3Signer | `%s` | `%s` |\n", r.Web3SignerDnpName, r.Web3SignerDnpVersion))
	sb.WriteString(fmt.Sprintf("| MEV Boost | `%s` | `%s` |\n", r.MevBoostDnpName, r.MevBoostDnpVersion))
	sb.WriteString(fmt.Sprintf("| Network | `%s` |  |\n", r.Network))
	sb.WriteString("\n")

	// Version tracking section
	sb.WriteString("### 🔖 Version Tracking\n\n")

	// Execution client versions
	sb.WriteString("**Execution Client Versions**\n\n")
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
	sb.WriteString("\n")

	// Consensus client versions
	sb.WriteString("**Consensus Client Versions**\n\n")
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
	sb.WriteString("\n")

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
	status := "FAILED"
	if r.Success {
		status = "PASSED"
	}
	sb.WriteString(fmt.Sprintf("%s - %s\n", r.reportTitle(), status))
	sb.WriteString("========================================\n\n")

	// Attestation section (if URLs present)
	if r.BeaconchainEpochURL != "" || len(r.BeaconchainValidatorURLs) > 0 {
		sb.WriteString("ATTESTATION:\n")
		if r.Success && r.BeaconchainEpochURL != "" {
			sb.WriteString(fmt.Sprintf("  - Epoch: %s\n", r.BeaconchainEpochURL))
		}
		for _, vurl := range r.BeaconchainValidatorURLs {
			sb.WriteString(fmt.Sprintf("  - Validator: %s\n", vurl))
		}
		sb.WriteString("\n")
	}

	// Clients Used
	sb.WriteString("CLIENTS USED:\n")
	sb.WriteString(fmt.Sprintf("  Execution:   %s", r.ExecutionDnpName))
	if r.ExecutionDnpVersion != "" {
		sb.WriteString(fmt.Sprintf(" (v%s)", r.ExecutionDnpVersion))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Consensus:   %s", r.ConsensusDnpName))
	if r.ConsensusDnpVersion != "" {
		sb.WriteString(fmt.Sprintf(" (v%s)", r.ConsensusDnpVersion))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Web3Signer:  %s", r.Web3SignerDnpName))
	if r.Web3SignerDnpVersion != "" {
		sb.WriteString(fmt.Sprintf(" (v%s)", r.Web3SignerDnpVersion))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  MEV Boost:   %s", r.MevBoostDnpName))
	if r.MevBoostDnpVersion != "" {
		sb.WriteString(fmt.Sprintf(" (v%s)", r.MevBoostDnpVersion))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Network:     %s\n", r.Network))
	sb.WriteString("\n")

	// Version tracking
	sb.WriteString("VERSION TRACKING:\n")

	// Execution client versions
	sb.WriteString("  Execution Client Versions:\n")
	if r.ExecutionClientVersionBefore != "" {
		sb.WriteString(fmt.Sprintf("    Before Install:     %s\n", r.ExecutionClientVersionBefore))
	} else {
		sb.WriteString("    Before Install:     _not available_\n")
	}
	if r.ExecutionClientVersionAfter != "" {
		sb.WriteString(fmt.Sprintf("    After Install:      %s\n", r.ExecutionClientVersionAfter))
	} else {
		sb.WriteString("    After Install:      _not available_\n")
	}

	// Consensus client versions
	sb.WriteString("  Consensus Client Versions:\n")
	if r.ConsensusClientVersionBefore != "" {
		sb.WriteString(fmt.Sprintf("    Before Install:     %s\n", r.ConsensusClientVersionBefore))
	} else {
		sb.WriteString("    Before Install:     _not available_\n")
	}
	if r.ConsensusClientVersionAfter != "" {
		sb.WriteString(fmt.Sprintf("    After Install:      %s\n", r.ConsensusClientVersionAfter))
	} else {
		sb.WriteString("    After Install:      _not available_\n")
	}
	sb.WriteString("\n")

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
