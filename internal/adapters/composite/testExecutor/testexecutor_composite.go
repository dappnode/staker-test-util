package testexecutor

import (
	"context"
	"fmt"
	"time"

	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/execution"
)

type TestExecutorAdapter struct {
	Execution   *execution.ExecutionAdapter
	Brain       *brain.BrainAdapter
	Beaconchain *beaconchain.BeaconchainAdapter
}

func NewTestExecutorAdapter(execution *execution.ExecutionAdapter, brain *brain.BrainAdapter, beaconchain *beaconchain.BeaconchainAdapter) *TestExecutorAdapter {
	return &TestExecutorAdapter{
		Execution:   execution,
		Brain:       brain,
		Beaconchain: beaconchain,
	}
}

// ExecuteTest waits for execution sync, gets validator indexes, and checks liveness
func (t *TestExecutorAdapter) ExecuteTest(ctx context.Context) error {
	// 1. Wait until execution is synced
	maxTries := 60
	for i := 0; i < maxTries; i++ {
		synced, err := t.Execution.GetIsSyncing(ctx)
		if err != nil {
			return fmt.Errorf("failed to check execution sync: %w", err)
		}
		if !synced {
			break
		}
		if i == maxTries-1 {
			return fmt.Errorf("execution client did not sync after %d attempts", maxTries)
		}
		time.Sleep(3 * time.Second)
	}

	// 2. Get validator pubkeys and indexes
	pubkeys, err := t.Brain.GetValidatorsPubkeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get validator pubkeys: %w", err)
	}
	if len(pubkeys) == 0 {
		return fmt.Errorf("no validator pubkeys found")
	}
	indexMap, err := t.Beaconchain.GetValidatorsIndexes(ctx, pubkeys)
	if err != nil {
		return fmt.Errorf("failed to get validator indexes: %w", err)
	}
	if len(indexMap) == 0 {
		return fmt.Errorf("no validator indexes found for pubkeys")
	}
	var indexes []string
	for _, idx := range indexMap {
		indexes = append(indexes, idx)
	}

	// 3. Wait for liveness up to 3 epochs (19m12s)
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
