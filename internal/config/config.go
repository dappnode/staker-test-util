package config

import (
	"clients-test/internal/logger"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all necessary configurations for adapters
type Config struct {
	modulePrefix   string
	BrainUrl       string
	ExecutionUrl   string
	BeaconchainUrl string
	DappmanagerUrl string
}

// LoadConfig loads environment variables and required files
func LoadConfig() (*Config, error) {
	// Load .env file if present (useful for local development)
	err := godotenv.Load()
	if err == nil {
		logger.DebugWithPrefix("CONFIG", "Loaded environment variables from .env file")
	}

	// Load required configurations
	config := &Config{
		modulePrefix:   "CONFIG",
		BrainUrl:       os.Getenv("BRAIN_URL"),
		ExecutionUrl:   os.Getenv("EXECUTION_URL"),
		BeaconchainUrl: os.Getenv("BEACONCHAIN_URL"),
		DappmanagerUrl: os.Getenv("DAPPMANAGER_URL"),
	}

	// Validate required fields
	if err := validateConfig(config); err != nil {
		logger.ErrorWithPrefix("CONFIG", "Configuration validation failed: %v", err)
		return nil, err
	}

	logger.InfoWithPrefix("CONFIG", "Configuration loaded successfully")
	return config, nil
}

// validateConfig ensures required fields are set
func validateConfig(cfg *Config) error {
	if cfg.BrainUrl == "" {
		logger.DebugWithPrefix(cfg.modulePrefix, "BRAIN_URL not set, using default value: http://brain.web3signer-holesky.dappnode:5000")
		cfg.BrainUrl = "http://brain.web3signer-holesky.dappnode:5000"
	}

	if cfg.ExecutionUrl == "" {
		logger.DebugWithPrefix(cfg.modulePrefix, "EXECUTION_URL not set, using default value: http://execution.holesky.dncore.dappnode:8545")
		cfg.ExecutionUrl = "http://execution.holesky.dncore.dappnode:8545"
	}

	if cfg.BeaconchainUrl == "" {
		logger.DebugWithPrefix(cfg.modulePrefix, "BEACONCHAIN_URL not set, using default value: http://beaconchain.holesky.dncore.dappnode:3500")
		cfg.BeaconchainUrl = "http://beaconchain.holesky.dncore.dappnode:3500"
	}

	if cfg.DappmanagerUrl == "" {
		logger.DebugWithPrefix(cfg.modulePrefix, "DAPPMANAGER_URL not set, using default value: http://my.dappnode:5000")
		cfg.DappmanagerUrl = "http://my.dappnode:5000"
	}

	return nil
}
