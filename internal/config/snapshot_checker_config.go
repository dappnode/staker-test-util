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
	ExecutionClient string
	Network         string
	CronIntervalSec int
	RunOnce         bool // If true, run once and exit (for testing)
}

// DefaultCronIntervalSec is 6 hours in seconds
const DefaultCronIntervalSec = 6 * 60 * 60

// ParseSnapshotCheckerConfig parses CLI flags and environment variables into a SnapshotCheckerAppConfig
func ParseSnapshotCheckerConfig() SnapshotCheckerAppConfig {
	// CLI flags
	executionClient := flag.String("execution-client", "", "Execution client name (geth, nethermind, reth, besu, erigon). Required.")
	network := flag.String("network", "hoodi", "Network name (e.g., hoodi). Default: hoodi")
	cronIntervalSec := flag.Int("cron-interval", 0, "Interval between snapshot checks in seconds. Default: 21600 (6 hours)")
	runOnce := flag.Bool("run-once", true, "Run once and exit (for testing). Default: false")

	flag.Parse()

	config := SnapshotCheckerAppConfig{
		ExecutionClient: strings.TrimSpace(strings.ToLower(getConfigValue(*executionClient, "EXECUTION_CLIENT", ""))),
		Network:         getConfigValue(*network, "NETWORK", "hoodi"),
		CronIntervalSec: getCronIntervalConfigValueInt(*cronIntervalSec, "CRON_INTERVAL_SEC", DefaultCronIntervalSec),
		RunOnce:         *runOnce || os.Getenv("RUN_ONCE") == "true",
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

// Validate checks that required configuration values are present and valid
func (c *SnapshotCheckerAppConfig) Validate() {
	// Validate execution client is specified
	if c.ExecutionClient == "" {
		logger.FatalWithPrefix(snapshotLogPrefix, "Execution client is required. Valid clients: %v", domain.ValidExecutionClients)
	}

	// Validate execution client is valid
	if !domain.IsValidExecutionClient(c.ExecutionClient) {
		logger.FatalWithPrefix(snapshotLogPrefix, "Invalid execution client '%s'. Valid clients: %v", c.ExecutionClient, domain.ValidExecutionClients)
	}

	// Validate cron interval
	if c.CronIntervalSec < 60 {
		logger.WarnWithPrefix(snapshotLogPrefix, "Cron interval %d seconds is very short, using minimum of 60 seconds", c.CronIntervalSec)
		c.CronIntervalSec = 60
	}

	logger.InfoWithPrefix(snapshotLogPrefix, "Configuration validated: network=%s, client=%s, interval=%ds, runOnce=%v",
		c.Network, c.ExecutionClient, c.CronIntervalSec, c.RunOnce)
}
