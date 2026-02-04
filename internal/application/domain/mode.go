package domain

import "fmt"

// RunMode represents the application's running mode
type RunMode string

const (
	// ModeSync only waits for execution and consensus clients to sync
	ModeSync RunMode = "sync"
	// ModeTest waits for sync and runs the full attestation test
	ModeTest RunMode = "test"
)

// ValidModes contains all valid run modes
var ValidModes = []RunMode{ModeSync, ModeTest}

// ParseRunMode converts a string to a RunMode, returning an error if invalid
func ParseRunMode(s string) (RunMode, error) {
	mode := RunMode(s)
	for _, valid := range ValidModes {
		if mode == valid {
			return mode, nil
		}
	}
	return "", fmt.Errorf("invalid mode '%s': must be one of %v", s, ValidModes)
}

// String returns the string representation of the RunMode
func (m RunMode) String() string {
	return string(m)
}

// IsSync returns true if the mode is sync-only
func (m RunMode) IsSync() bool {
	return m == ModeSync
}

// IsTest returns true if the mode is full test
func (m RunMode) IsTest() bool {
	return m == ModeTest
}
