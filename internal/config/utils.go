package config

import "os"

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
