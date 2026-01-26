package composite

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/apis/snapshots"
	"clients-test/internal/adapters/composite/cleaner"
	"clients-test/internal/adapters/composite/ensurer"
	"clients-test/internal/adapters/composite/executor"
	"clients-test/internal/adapters/shared/blocknumber"
	"clients-test/internal/application/domain"
	"context"
	"time"
)

type CompositeAdapter struct {
	ensurer  *ensurer.EnsurerAdapter
	executor *executor.ExecutorAdapter
	cleaner  *cleaner.CleanerAdapter
	docker   *docker.DockerAdapter
	github   *github.GitHubAdapter
	report   *domain.TestReport
}

func NewCompositeAdapter(
	dappManagerAdapter *dappmanager.DappManagerAdapter,
	brainAdapter *brain.BrainAdapter,
	dockerAdapter *docker.DockerAdapter,
	snapshotsAdapter *snapshots.SnapshotsAdapter,
	beaconchainAdapter *beaconchain.BeaconchainAdapter,
	executionAdapter *execution.ExecutionAdapter,
	ipfsAdapter *ipfs.IPFSAdapter,
	githubAdapter *github.GitHubAdapter,
	blockNumberAdapter *blocknumber.BlockNumberAdapter,
) *CompositeAdapter {
	ensurer := ensurer.NewEnsurerAdapter(dappManagerAdapter, brainAdapter, dockerAdapter, snapshotsAdapter, beaconchainAdapter, executionAdapter, ipfsAdapter, blockNumberAdapter)
	executor := executor.NewExecutorAdapter(executionAdapter, brainAdapter, beaconchainAdapter)
	cleaner := cleaner.NewCleanerAdapter(dappManagerAdapter, executionAdapter, brainAdapter, beaconchainAdapter, dockerAdapter)
	return &CompositeAdapter{
		ensurer:  ensurer,
		executor: executor,
		cleaner:  cleaner,
		docker:   dockerAdapter,
		github:   githubAdapter,
	}
}

func (t *CompositeAdapter) EnsureEnvironment(ctx context.Context, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// Initialize the report
	t.report = domain.NewTestReport(stakerConfig)

	return t.ensurer.EnsureEnvironment(ctx, stakerConfig, pkg, t.report)
}

func (t *CompositeAdapter) ExecuteTest(ctx context.Context, stakerConfig domain.StakerConfig) error {
	// Record test start time for log collection
	testStartTime := time.Now()

	// Run the actual test
	testErr := t.executor.ExecuteTest(ctx, t.report)

	// Record test end time
	testEndTime := time.Now()

	// Collect container error logs from all relevant containers
	t.collectContainerErrorLogs(ctx, stakerConfig, testStartTime, testEndTime)

	// Set the final result
	t.report.SetResult(testErr == nil, testErr)

	// Comment on PR if GitHub integration is enabled (ignore errors - don't fail test for PR comment issues)
	_ = t.github.CommentOnPR(ctx, t.report)

	return testErr
}

func (t *CompositeAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig) error {
	return t.cleaner.CleanEnvironment(ctx, stakerConfig)
}

// collectContainerErrorLogs collects error logs from all relevant containers
func (t *CompositeAdapter) collectContainerErrorLogs(ctx context.Context, stakerConfig domain.StakerConfig, since, until time.Time) {
	const maxLinesPerContainer = 3

	containerNames := []string{
		stakerConfig.BrainContainerName,
		stakerConfig.SignerContainerName,
		stakerConfig.BeaconchainContainerName,
		stakerConfig.ValidatorContainerName,
		stakerConfig.ExecutionContainerName,
	}

	errorLogs := t.docker.CollectAllContainerErrorLogs(ctx, containerNames, since, until, maxLinesPerContainer)

	for containerName, lines := range errorLogs {
		t.report.AddContainerErrors(containerName, lines)
	}
}
