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

// waitForExecutionSync waits until the execution client is synced or times out
func (t *ExecutorAdapter) waitForExecutionSync(ctx context.Context) error {
	maxTries := 60
	for i := 0; i < maxTries; i++ {
		synced, err := t.Execution.GetIsSyncing(ctx)
		if err != nil {
			return fmt.Errorf("failed to check execution sync: %w", err)
		}
		if !synced {
			return nil
		}
		if i == maxTries-1 {
			return fmt.Errorf("execution client did not sync after %d attempts", maxTries)
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}

// waitForValidatorLiveness waits for all validators to become live up to 3 epochs
func (t *ExecutorAdapter) waitForValidatorLiveness(ctx context.Context) error {
	pubkeys, err := t.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch validators from brain: %v", err)
	}
	if len(pubkeys) == 0 {
		return fmt.Errorf("at least 1 validator must be loaded to be able to run the test")
	}
	indexes, err := t.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
	if err != nil {
		return fmt.Errorf("failed to get validators indexes: %w", err)
	}
	if len(indexes) == 0 {
		return fmt.Errorf("no validator indexes provided")
	}
	maxEpochs := 3
	epochDuration := 6*time.Minute + 24*time.Second // 384 seconds
	for epoch := 0; epoch < maxEpochs; epoch++ {
		liveness, err := t.Beaconchain.GetValidatorLiveness(ctx, indexes)
		if err != nil {
			return fmt.Errorf("failed to get validator liveness: %w", err)
		}
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
		if epoch == maxEpochs-1 {
			return fmt.Errorf("validators did not become live after %d epochs", maxEpochs)
		}
		time.Sleep(epochDuration)
	}
	return nil
}

// ExecuteTest runs both sync and liveness checks in sequence
func (t *ExecutorAdapter) ExecuteTest(ctx context.Context) error {
	if err := t.waitForExecutionSync(ctx); err != nil {
		return err
	}
	if err := t.waitForValidatorLiveness(ctx); err != nil {
		return err
	}
	return nil
}
