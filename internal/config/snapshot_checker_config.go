package config

import (
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"flag"
	"os"
	"strings"
)

var snapshotLogPrefix = "SnapshotCheckerConfig"

// SnapshotCheckerAppConfig holds all application configuration for snapshot checker
type SnapshotCheckerAppConfig struct {
	ExecutionClients []string
	Network          string
	CronIntervalSec  int
	RunOnce          bool // If true, run once and exit (for testing)
}

// DefaultCronIntervalSec is 6 hours in seconds
const DefaultCronIntervalSec = 6 * 60 * 60

// ParseSnapshotCheckerConfig parses CLI flags and environment variables into a SnapshotCheckerAppConfig
func ParseSnapshotCheckerConfig() SnapshotCheckerAppConfig {
	// CLI flags
	executionClients := flag.String("execution-clients", "", "Comma-separated list of execution clients (geth, nethermind, reth, besu, erigon). Default: all")
	network := flag.String("network", "hoodi", "Network name (e.g., hoodi). Default: hoodi")
	cronIntervalSec := flag.Int("cron-interval", 0, "Interval between snapshot checks in seconds. Default: 21600 (6 hours)")
	runOnce := flag.Bool("run-once", false, "Run once and exit (for testing). Default: false")

	flag.Parse()

	config := SnapshotCheckerAppConfig{
		ExecutionClients: parseExecutionClients(getConfigValue(*executionClients, "EXECUTION_CLIENTS", "")),
		Network:          getConfigValue(*network, "NETWORK", "hoodi"),
		CronIntervalSec:  getCronIntervalConfigValueInt(*cronIntervalSec, "CRON_INTERVAL_SEC", DefaultCronIntervalSec),
		RunOnce:          *runOnce || os.Getenv("RUN_ONCE") == "true",
	}

	return config
}

// getCronIntervalConfigValueInt returns the flag value if set, otherwise falls back to the environment variable, then default
func getCronIntervalConfigValueInt(flagValue int, envName string, defaultValue int) int {
	if flagValue != 0 {
		return flagValue
	}
	if envValue := os.Getenv(envName); envValue != "" {
		var result int
		for _, c := range envValue {
			if c >= '0' && c <= '9' {
				result = result*10 + int(c-'0')
			}
		}
		if result > 0 {
			return result
		}
	}
	return defaultValue
}

// parseExecutionClients parses the comma-separated list of execution clients
// Returns empty slice if input is empty (means all clients)
func parseExecutionClients(input string) []string {
	if input == "" {
		return nil // nil means all clients
	}

	var clients []string
	for _, client := range strings.Split(input, ",") {
		client = strings.TrimSpace(strings.ToLower(client))
		if client != "" {
			clients = append(clients, client)
		}
	}
	return clients
}

// Validate checks that required configuration values are present and valid
func (c *SnapshotCheckerAppConfig) Validate() {
	// Validate execution clients if specified
	if len(c.ExecutionClients) > 0 {
		for _, client := range c.ExecutionClients {
			if !domain.IsValidExecutionClient(client) {
				logger.FatalWithPrefix(snapshotLogPrefix, "Invalid execution client '%s'. Valid clients: %v", client, domain.ValidExecutionClients)
			}
		}
	}

	// Validate cron interval
	if c.CronIntervalSec < 60 {
		logger.WarnWithPrefix(snapshotLogPrefix, "Cron interval %d seconds is very short, using minimum of 60 seconds", c.CronIntervalSec)
		c.CronIntervalSec = 60
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "Configuration validated: network=%s, clients=%v, interval=%ds, runOnce=%v",
		c.Network, c.ExecutionClients, c.CronIntervalSec, c.RunOnce)
}
