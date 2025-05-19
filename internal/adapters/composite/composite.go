package composite

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/tropidatooor"
	"clients-test/internal/adapters/composite/cleaner"
	"clients-test/internal/adapters/composite/ensurer"
	"clients-test/internal/adapters/composite/executor"
	"clients-test/internal/adapters/system/mount"
	"clients-test/internal/application/domain"
	"context"
)

type CompositeAdapter struct {
	ensurer  *ensurer.EnsurerAdapter
	executor *executor.ExecutorAdapter
	cleaner  *cleaner.CleanerAdapter
}

func NewCompositeAdapter(
	dappManagerAdapter *dappmanager.DappManagerAdapter,
	brainAdapter *brain.BrainAdapter,
	tropidatooorAdapter *tropidatooor.TropidatooorAdapter,
	dockerAdapter *docker.DockerAdapter,
	mountAdapter *mount.MountAdapter,
	beaconchainAdapter *beaconchain.BeaconchainAdapter,
	executionAdapter *execution.ExecutionAdapter,
	ipfsAdapter *ipfs.IPFSAdapter,
) *CompositeAdapter {
	ensurer := ensurer.NewEnsurerAdapter(dappManagerAdapter, brainAdapter, tropidatooorAdapter, dockerAdapter, mountAdapter, beaconchainAdapter, executionAdapter, ipfsAdapter)
	executor := executor.NewExecutorAdapter(executionAdapter, brainAdapter, beaconchainAdapter)
	cleaner := cleaner.NewCleanerAdapter(dappManagerAdapter, executionAdapter, brainAdapter, beaconchainAdapter, dockerAdapter, mountAdapter)
	return &CompositeAdapter{ensurer, executor, cleaner}
}

func (t *CompositeAdapter) EnsureEnvironment(ctx context.Context, mountConfig domain.Mount, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	return t.ensurer.EnsureEnvironment(ctx, mountConfig, stakerConfig, pkg)
}

func (t *CompositeAdapter) ExecuteTest(ctx context.Context) error {
	return t.executor.ExecuteTest(ctx)
}

func (t *CompositeAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, mountConfig domain.Mount) error {
	return t.cleaner.CleanEnvironment(ctx, stakerConfig, mountConfig)
}
