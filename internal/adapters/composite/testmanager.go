package composite

import (
	"clients-test/internal/adapters/apis/beaconchain"
	"clients-test/internal/adapters/apis/brain"
	"clients-test/internal/adapters/apis/dappmanager"
	"clients-test/internal/adapters/apis/docker"
	"clients-test/internal/adapters/apis/execution"
	"clients-test/internal/adapters/apis/github"
	"clients-test/internal/adapters/apis/ipfs"
	"clients-test/internal/adapters/composite/cleaner"
	"clients-test/internal/adapters/composite/ensurer"
	"clients-test/internal/adapters/composite/executor"
	"clients-test/internal/application/domain"
	"clients-test/internal/logger"
	"context"
	"fmt"
	"time"
)

type TestManagerAdapter struct {
	ensurer  *ensurer.EnsurerAdapter
	executor *executor.ExecutorAdapter
	cleaner  *cleaner.CleanerAdapter
	docker   *docker.DockerAdapter
	github   *github.GitHubAdapter
	report   *domain.TestReport
}

func NewTestManagerAdapter(
	stakerConfig domain.StakerConfig,
	dockerAdapter *docker.DockerAdapter,
	ipfsAdapter *ipfs.IPFSAdapter,
	githubAdapter *github.GitHubAdapter,
) *TestManagerAdapter {
	dappManagerAdapter := dappmanager.NewDappManagerAdapter()
	brainAdapter := brain.NewBrainAdapter(stakerConfig.Urls.BrainURL)
	beaconchainAdapter := beaconchain.NewBeaconchainAdapter(stakerConfig.Urls.BeaconchainURL)
	executionAdapter := execution.NewExecutionAdapter(stakerConfig.Urls.ExecutionURL)

	ensurer := ensurer.NewEnsurerAdapter(dappManagerAdapter, brainAdapter, dockerAdapter, beaconchainAdapter, executionAdapter, ipfsAdapter)
	executor := executor.NewExecutorAdapter(executionAdapter, brainAdapter, beaconchainAdapter)
	cleaner := cleaner.NewCleanerAdapter(dappManagerAdapter, executionAdapter, dockerAdapter)
	return &TestManagerAdapter{
		ensurer:  ensurer,
		executor: executor,
		cleaner:  cleaner,
		docker:   dockerAdapter,
		github:   githubAdapter,
	}
}

func (t *TestManagerAdapter) EnsureEnvironment(ctx context.Context, mode domain.RunMode, stakerConfig domain.StakerConfig, pkg domain.Pkg) error {
	// Initialize the report
	t.report = domain.NewTestReport(stakerConfig)

	return t.ensurer.EnsureEnvironment(ctx, mode, stakerConfig, pkg, t.report)
}

func (t *TestManagerAdapter) ExecuteSync(ctx context.Context, stakerConfig domain.StakerConfig) error {
	// Record start time for log collection
	startTime := time.Now()

	// Run the sync-only operation
	syncErr := t.executor.ExecuteSync(ctx, t.report)

	// Record end time
	endTime := time.Now()

	// Collect container error logs from all relevant containers
	t.collectContainerErrorLogs(ctx, stakerConfig, startTime, endTime)

	// Set the final result
	t.report.SetResult(syncErr == nil, syncErr)

	// Print the report to the console
	fmt.Println(t.report.ToConsoleString())

	// Comment on PR if GitHub integration is enabled (ignore errors - don't fail for PR comment issues)
	err := t.github.CommentOnPR(ctx, t.report)
	if err != nil {
		logger.Error("Failed to comment on PR: %v", err)
	}

	return syncErr
}

func (t *TestManagerAdapter) ExecuteTest(ctx context.Context, stakerConfig domain.StakerConfig) error {
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

	// Print the report to the console
	fmt.Println(t.report.ToConsoleString())

	// Comment on PR if GitHub integration is enabled (ignore errors - don't fail test for PR comment issues)
	err := t.github.CommentOnPR(ctx, t.report)
	if err != nil {
		logger.Error("Failed to comment on PR: %v", err)
	}

	return testErr
}

func (t *TestManagerAdapter) CleanEnvironment(ctx context.Context, stakerConfig domain.StakerConfig) error {
	return t.cleaner.CleanEnvironment(ctx, stakerConfig)
}

// collectContainerErrorLogs collects error logs from all relevant containers
func (t *TestManagerAdapter) collectContainerErrorLogs(ctx context.Context, stakerConfig domain.StakerConfig, since, until time.Time) {
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
