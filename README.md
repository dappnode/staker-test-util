# Dappnode ethereum clients test SDK

A testing utility for Dappnode staker packages that runs in GitHub self-hosted runners. It can execute tests either manually or automatically on GitHub pull requests, with automatic PR commenting for test reports.

## Features

- **Automated Testing**: Tests staker packages (execution, consensus, web3signer, etc.)
- **GitHub Integration**: Automatically comments test reports on pull requests
- **Timing Measurements**: Measures duration of all test phases (setup, execution, cleanup)
- **Container Log Collection**: Captures error logs from all relevant containers during test execution
- **Detailed Reports**: Generates comprehensive reports with clients used, timings, and error logs

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `IPFS_GATEWAY_URL` | IPFS gateway URL for fetching packages |
| `IPFS_HASH` | IPFS hash of the test package |

### GitHub Integration (Optional)

These environment variables enable automatic PR commenting. When running in GitHub Actions, most of these are set automatically.

| Variable | Description | GitHub Actions Auto-set |
|----------|-------------|------------------------|
| `GITHUB_TOKEN` | GitHub token with PR comment permissions | ✅ (via `${{ secrets.GITHUB_TOKEN }}`) |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format | ✅ |
| `GITHUB_PR_NUMBER` | Pull request number | ❌ (extract from event) |
| `GITHUB_RUN_ID` | GitHub Actions run ID (for linking to logs) | ✅ |
| `GITHUB_SERVER_URL` | GitHub server URL (default: `https://github.com`) | ✅ |

### CLI Flags

All environment variables can also be set via CLI flags:

```bash
./staker-test-util \
  --ipfs-gateway-url="http://ipfs.dappnode:8080" \
  --ipfs-hash="QmSfPFSauovbMzEcvf2a2csoHtfqpViShwEYpuX3fPR8zv" \
  --github-token="ghp_xxxx" \
  --github-repository="dappnode/staker-test-util" \
  --github-pr-number="123" \
  --github-run-id="12345678" \
  --github-server-url="https://github.com"
```

## GitHub Actions Workflow Example

```yaml
name: Staker Test

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  test:
    runs-on: self-hosted
    steps:
      - name: Run Staker Test
        env:
          IPFS_GATEWAY_URL: http://ipfs.dappnode:8080
          IPFS_HASH: ${{ github.event.inputs.ipfs_hash }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPOSITORY: ${{ github.repository }}
          GITHUB_PR_NUMBER: ${{ github.event.pull_request.number }}
          GITHUB_RUN_ID: ${{ github.run_id }}
          GITHUB_SERVER_URL: ${{ github.server_url }}
        run: |
          docker-compose up --build
```

## Test Report

The test report includes:

### Clients Used
- Execution client DNP name
- Consensus client DNP name
- Web3Signer DNP name
- MEV Boost DNP name
- Network

### Timing Measurements

#### Environment Setup
- SetStakerConfig
- PackageInstall
- StopAndGetVolumeTarget
- DownloadAndExtractSnapshot
- StartContainer

#### Test Execution
- WaitForBeaconchainSync
- WaitForExecutionSync
- WaitForValidatorLiveness

### Container Error Logs
Captures error lines from:
- Brain container
- Signer container
- Beaconchain container
- Validator container
- Execution container

> Shows up to 3 error lines per container. Full logs available in CI.

## Docker Compose

```bash
# With GitHub integration
GITHUB_TOKEN=ghp_xxxx \
GITHUB_REPOSITORY=dappnode/staker-test-util \
GITHUB_PR_NUMBER=123 \
docker-compose up --build
```

## TODO's
- [x] Implement a github adapter to interact with issues and PRs so we can automate report creation as well as the testing process.
- [x] Measure the time it takes every process in the test and add it to the report
- [x] Collect logs from containers and create a report
- [x] Add to the report the clients used
- [ ] Auto-updates for this dappnode must run much more often than production, so clients are always updated to latest versions
- [ ] Research how to release this SDK tool to be run from a github action directly
- [ ] Implement when manual trigger (`workflow_dispatch`) the clients will be passed as inputs, use them to create the staker config for the test
- [ ] Implement edit of `/usr/src/dappnode/DNCORE/docker-compose.yml` file to add env `TEST=true` and relaunch compose
- [ ] Print version of the clients 
    - [ ] For the EC it must be printed
- [ ] Consider always removing beacon volumes to ensure avoiding old states of chain and always start with the checkpoint sync
- [ ] Implement switch off of dappmanager cron that restarts containers of clients selected in stakers
- [ ] Silent the tar output when extracting snapshots or make it less verbose
- [ ] Consider adding to report beaconcha validator url
- [ ] Consider setting keystore and password through github secrets.