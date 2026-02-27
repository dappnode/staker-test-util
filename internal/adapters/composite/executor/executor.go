package executor

import (
	"context"
	"fmt"
	"time"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
)

type ExecutorAdapter struct {
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
}

func NewExecutorAdapter(execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter) *ExecutorAdapter {
	return &ExecutorAdapter{
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
	}
}

// timeOperation measures the duration of an operation and records it in the report
func timeOperation(report *domain.TestReport, operationName string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	success := err == nil
	report.AddExecuteTiming(operationName, duration, success, err)

	return err
}

// waitForExecutionSync waits until the execution client is synced or times out,
// returning only after maxTries with the most recent error (if any).
func (t *ExecutorAdapter) waitForExecutionSync(ctx context.Context) error {
	const (
		maxTries = 180
		sleepDur = 6 * time.Second
	)
	var lastErr error

	for i := 0; i < maxTries; i++ {
		logger.Info("[ExecutionSync] Attempt %d/%d: Checking execution client sync status...", i+1, maxTries)
		syncing, err := t.Execution.GetIsSyncing(ctx)
		if err != nil {
			// record the error, but don't bail out yet
			lastErr = fmt.Errorf("check execution sync attempt %d failed: %w", i+1, err)
			logger.Error("Execution sync check failed (attempt %d): %v", i+1, err)
		} else if !syncing {
			logger.Info("[ExecutionSync] Execution client is synced (attempt %d)", i+1)
			// once we see “not syncing” we know the node is caught up
			return nil
		} else {
			logger.Info("[ExecutionSync] Execution client still syncing (attempt %d)", i+1)
		}

		// if we're on the last try, break and return lastErr
		if i < maxTries-1 {
			time.Sleep(sleepDur)
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("execution client did not sync after %d attempts", maxTries)
}

// waitForBeaconchainSync waits for the beacon chain to be synced.
// It retries fetching the sync status up to maxTries, returning the last error encountered.
func (t *ExecutorAdapter) waitForBeaconchainSync(ctx context.Context) error {
	const (
		maxTries = 60
		sleepDur = 6 * time.Second
	)
	var lastErr error

	for i := 0; i < maxTries; i++ {
		logger.Info("[BeaconchainSync] Attempt %d/%d: Checking beaconchain sync status...", i+1, maxTries)
		syncing, err := t.Beaconchain.GetIsSyncing(ctx)
		if err != nil {
			lastErr = fmt.Errorf("check beaconchain sync attempt %d failed: %w", i+1, err)
			logger.Error("Beaconchain sync check failed (attempt %d): %v", i+1, err)
		} else if !syncing {
			logger.Info("[BeaconchainSync] Beaconchain is synced (attempt %d)", i+1)
			return nil // synced!
		} else {
			logger.Info("[BeaconchainSync] Beaconchain still syncing (attempt %d)", i+1)
		}

		if i < maxTries-1 {
			time.Sleep(sleepDur)
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("beaconchain did not sync after %d attempts", maxTries)
}

// waitForValidatorLiveness waits for all validators to become live up to maxEpochs.
// It only errors out after maxEpochs, returning the last error encountered.
func (t *ExecutorAdapter) waitForValidatorLiveness(ctx context.Context, report *domain.TestReport) error {
	const (
		maxSlots    = 32 * 5 // 384s
		slotSeconds = 12
	)
	slotDuration := time.Duration(slotSeconds) * time.Second

	var lastErr error

	// First, we need pubkeys and indexes—retry these as well.
	var pubkeys []string
	for i := 0; i < maxSlots; i++ {
		logger.Info("[ValidatorLiveness] Attempt %d/%d: Fetching validator pubkeys...", i+1, maxSlots)
		var err error
		pubkeys, err = t.Brain.GetValidatorsPubkeys(ctx)
		if err != nil {
			lastErr = fmt.Errorf("fetch validators attempt %d failed: %w", i+1, err)
			logger.Error("Fetch validators pubkeys failed (attempt %d): %v", i+1, err)
		} else if len(pubkeys) == 0 {
			lastErr = fmt.Errorf("attempt %d: no validators loaded", i+1)
			logger.Error("No validators loaded (attempt %d)", i+1)
		} else {
			logger.Info("[ValidatorLiveness] Got %d validator pubkeys (attempt %d)", len(pubkeys), i+1)
			break
		}
		if i < maxSlots-1 {
			time.Sleep(slotDuration)
		}
	}
	if len(pubkeys) == 0 {
		return lastErr
	}

	var indexes []string
	for i := 0; i < maxSlots; i++ {
		logger.Info("[ValidatorLiveness] Attempt %d/%d: Fetching validator indexes...", i+1, maxSlots)
		var err error
		indexes, err = t.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
		if err != nil {
			lastErr = fmt.Errorf("get validator indexes attempt %d failed: %w", i+1, err)
			logger.Error("Get validator indexes failed (attempt %d): %v", i+1, err)
		} else if len(indexes) == 0 {
			lastErr = fmt.Errorf("attempt %d: no validator indexes returned", i+1)
			logger.Error("No validator indexes returned (attempt %d)", i+1)
		} else {
			logger.Info("[ValidatorLiveness] Got %d validator indexes (attempt %d)", len(indexes), i+1)
			logger.Info("[ValidatorLiveness] Validator indexes: %v", indexes)
			break
		}
		if i < maxSlots-1 {
			time.Sleep(slotDuration)
		}
	}
	if len(indexes) == 0 {
		return lastErr
	}

	// Set validator URLs in report
	validatorURLs := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		validatorURLs = append(validatorURLs, fmt.Sprintf("https://hoodi.beaconcha.in/validator/%s", idx))
	}
	report.BeaconchainValidatorURLs = validatorURLs

	// Now poll liveness over up to maxSlots
	for slot := 0; slot < maxSlots; slot++ {
		logger.Info("[ValidatorLiveness] Slot %d/%d: Checking validator liveness...", slot+1, maxSlots)
		liveness, epoch, err := t.Beaconchain.GetValidatorLiveness(ctx, indexes)
		if err != nil {
			lastErr = fmt.Errorf("get validator liveness slot %d failed: %w", slot, err)
			logger.Error("Get validator liveness failed (slot %d): %v", slot, err)
		} else {
			// Set epoch URL in report (only set once, first successful call)
			if report.BeaconchainEpochURL == "" && epoch > 0 {
				report.BeaconchainEpochURL = fmt.Sprintf("https://hoodi.beaconcha.in/epoch/%d", epoch)
			}
			allLive := true
			for _, live := range liveness {
				if !live {
					allLive = false
					break
				}
			}
			if allLive {
				logger.Info("[ValidatorLiveness] All validators are live at slot %d", slot+1)
				return nil
			}
			lastErr = fmt.Errorf("slot %d: some validators still not live", slot)
			logger.Info("Some validators still not live (slot %d)", slot)
		}

		if slot < maxSlots-1 {
			time.Sleep(slotDuration)
		}
	}

	return lastErr
}

// ExecuteSync runs only the sync checks (beaconchain and execution)
// All operations are timed and recorded in the report.
func (t *ExecutorAdapter) ExecuteSync(ctx context.Context, report *domain.TestReport) error {
	if err := timeOperation(report, "WaitForBeaconchainSync", func() error {
		return t.waitForBeaconchainSync(ctx)
	}); err != nil {
		return err
	}

	if err := timeOperation(report, "WaitForExecutionSync", func() error {
		return t.waitForExecutionSync(ctx)
	}); err != nil {
		return err
	}

	return nil
}

// ExecuteTest runs both sync and liveness checks in sequence
// All operations are timed and recorded in the report.
func (t *ExecutorAdapter) ExecuteTest(ctx context.Context, report *domain.TestReport) error {
	if err := timeOperation(report, "WaitForBeaconchainSync", func() error {
		return t.waitForBeaconchainSync(ctx)
	}); err != nil {
		return err
	}

	if err := timeOperation(report, "WaitForExecutionSync", func() error {
		return t.waitForExecutionSync(ctx)
	}); err != nil {
		return err
	}

	if err := timeOperation(report, "WaitForValidatorLiveness", func() error {
		return t.waitForValidatorLiveness(ctx, report)
	}); err != nil {
		return err
	}

	return nil
}
