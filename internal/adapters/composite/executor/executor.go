package executor

import (
	"context"
	"fmt"
	"time"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
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

// waitForExecutionSync waits until the execution client is synced or times out,
// returning only after maxTries with the most recent error (if any).
func (t *ExecutorAdapter) waitForExecutionSync(ctx context.Context) error {
	const (
		maxTries = 60
		sleepDur = 3 * time.Second
	)
	var lastErr error

	for i := 0; i < maxTries; i++ {
		syncing, err := t.Execution.GetIsSyncing(ctx)
		if err != nil {
			// record the error, but don't bail out yet
			lastErr = fmt.Errorf("check execution sync attempt %d failed: %w", i+1, err)
		} else if !syncing {
			// once we see “not syncing” we know the node is caught up
			return nil
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
		sleepDur = 3 * time.Second
	)
	var lastErr error

	for i := 0; i < maxTries; i++ {
		syncing, err := t.Beaconchain.GetIsSyncing(ctx)
		if err != nil {
			lastErr = fmt.Errorf("check beaconchain sync attempt %d failed: %w", i+1, err)
		} else if !syncing {
			return nil // synced!
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
func (t *ExecutorAdapter) waitForValidatorLiveness(ctx context.Context) error {
	const (
		maxEpochs    = 5
		epochSeconds = 6*60 + 24 // 384s
	)
	epochDuration := time.Duration(epochSeconds) * time.Second

	var lastErr error

	// First, we need pubkeys and indexes—retry these as well.
	var pubkeys []string
	for i := 0; i < maxEpochs; i++ {
		var err error
		pubkeys, err = t.Brain.GetValidatorsPubkeys(ctx)
		if err != nil {
			lastErr = fmt.Errorf("fetch validators attempt %d failed: %w", i+1, err)
		} else if len(pubkeys) == 0 {
			lastErr = fmt.Errorf("attempt %d: no validators loaded", i+1)
		} else {
			break
		}
		if i < maxEpochs-1 {
			time.Sleep(epochDuration)
		}
	}
	if len(pubkeys) == 0 {
		return lastErr
	}

	var indexes []string
	for i := 0; i < maxEpochs; i++ {
		var err error
		indexes, err := t.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
		if err != nil {
			lastErr = fmt.Errorf("get validator indexes attempt %d failed: %w", i+1, err)
		} else if len(indexes) == 0 {
			lastErr = fmt.Errorf("attempt %d: no validator indexes returned", i+1)
		} else {
			break
		}
		if i < maxEpochs-1 {
			time.Sleep(epochDuration)
		}
	}
	if len(indexes) == 0 {
		return lastErr
	}

	// Now poll liveness over up to maxEpochs
	for epoch := 0; epoch < maxEpochs; epoch++ {
		liveness, err := t.Beaconchain.GetValidatorLiveness(ctx, indexes)
		if err != nil {
			lastErr = fmt.Errorf("get validator liveness epoch %d failed: %w", epoch, err)
		} else {
			allLive := true
			for _, live := range liveness {
				if !live {
					allLive = false
					break
				}
			}
			if allLive {
				return nil
			}
			lastErr = fmt.Errorf("epoch %d: some validators still not live", epoch)
		}

		if epoch < maxEpochs-1 {
			time.Sleep(epochDuration)
		}
	}

	return lastErr
}

// ExecuteTest runs both sync and liveness checks in sequence
func (t *ExecutorAdapter) ExecuteTest(ctx context.Context) error {
	if err := t.waitForExecutionSync(ctx); err != nil {
		return err
	}
	if err := t.waitForBeaconchainSync(ctx); err != nil {
		return err
	}
	if err := t.waitForValidatorLiveness(ctx); err != nil {
		return err
	}
	return nil
}
