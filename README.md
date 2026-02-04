# Dappnode ethereum clients test SDK

A testing utility for Dappnode staker packages that runs in GitHub self-hosted runners. It can execute tests either manually or automatically on GitHub pull requests, with automatic PR commenting for test reports.

## Features

- **Two Run Modes**: Support for `sync` mode (wait for sync only) and `test` mode (full attestation test)
- **Automated Testing**: Tests staker packages (execution, consensus, web3signer, etc.)
- **GitHub Integration**: Automatically comments test reports on pull requests
- **Timing Measurements**: Measures duration of all test phases (setup, execution, cleanup)
- **Container Log Collection**: Captures error logs from all relevant containers during test execution
- **Detailed Reports**: Generates comprehensive reports with clients used, timings, and error logs

## Run Modes

The application supports two run modes, specified via the mandatory `--mode` flag or `MODE` environment variable:

### Sync Mode (`--mode=sync`)
- Waits for execution and consensus clients to sync
- Does NOT run the attestation/validator liveness test
- Useful for just ensuring nodes are synced before other operations

### Test Mode (`--mode=test`)
- Waits for execution and consensus clients to sync
- Runs the full attestation test (validator liveness check)
- This is the full testing workflow

## Quick Start (CI Usage)

### Using Docker Images from GHCR

Docker images are automatically published to GitHub Container Registry on every push to main and on releases.

```bash
ghcr.io/dappnode/staker-test-util/test-runner:latest
```

### Sync Mode in GitHub Actions

```yaml
- name: Sync Ethereum Clients
  run: |
    docker run --rm \
      --network dncore_network \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /var/lib/docker/volumes:/var/lib/docker/volumes \
      -e MODE=sync \
      -e IPFS_GATEWAY_URL=http://ipfs.dappnode:8080 \
      -e IPFS_HASH=${{ env.IPFS_HASH }} \
      -e EXECUTION_CLIENT=geth \
      -e CONSENSUS_CLIENT=nimbus \
      ghcr.io/dappnode/staker-test-util/test-runner:latest
```

### Test Mode in GitHub Actions

```yaml
- name: Run Staker Tests
  run: |
    docker run --rm \
      --network dncore_network \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /var/lib/docker/volumes:/var/lib/docker/volumes \
      -e MODE=test \
      -e IPFS_GATEWAY_URL=http://ipfs.dappnode:8080 \
      -e IPFS_HASH=${{ env.IPFS_HASH }} \
      -e EXECUTION_CLIENT=geth \
      -e CONSENSUS_CLIENT=nimbus \
      -e GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }} \
      -e GITHUB_REPOSITORY=${{ github.repository }} \
      -e GITHUB_PR_NUMBER=${{ github.event.pull_request.number }} \
      ghcr.io/dappnode/staker-test-util/test-runner:latest
```

See [examples/workflows/](examples/workflows/) for complete workflow examples.

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `MODE` | Run mode: `sync` or `test` | **Yes** |
| `IPFS_GATEWAY_URL` | IPFS gateway URL for fetching packages | Yes |
| `IPFS_HASH` | IPFS hash of the test package | Yes |
| `EXECUTION_CLIENT` | Override execution client (geth, reth, nethermind, besu, erigon) | No |
| `CONSENSUS_CLIENT` | Override consensus client (prysm, teku, nimbus, lodestar) | No |
| `LOG_LEVEL` | Log level (DEBUG, INFO, WARN, ERROR) | No |

> **Note:** If the package being tested is an execution or consensus client, it will override the respective environment variable with a warning.

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
# Sync Mode
./test-runner \
  --mode=sync \
  --ipfs-gateway-url="http://ipfs.dappnode:8080" \
  --ipfs-hash="QmSfPFSauovbMzEcvf2a2csoHtfqpViShwEYpuX3fPR8zv" \
  --execution-client="geth" \
  --consensus-client="nimbus"

# Test Mode
./test-runner \
  --mode=test \
  --ipfs-gateway-url="http://ipfs.dappnode:8080" \
  --ipfs-hash="QmSfPFSauovbMzEcvf2a2csoHtfqpViShwEYpuX3fPR8zv" \
  --execution-client="geth" \
  --consensus-client="nimbus" \
  --github-token="ghp_xxxx" \
  --github-repository="dappnode/staker-test-util" \
  --github-pr-number="123"
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

#### Execution (varies by mode)

**Sync Mode:**
- WaitForBeaconchainSync
- WaitForExecutionSync

**Test Mode:**
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
# Sync mode with GitHub integration
MODE=sync \
GITHUB_TOKEN=ghp_xxxx \
GITHUB_REPOSITORY=dappnode/staker-test-util \
GITHUB_PR_NUMBER=123 \
docker-compose up --build

# Test mode with GitHub integration
MODE=test \
GITHUB_TOKEN=ghp_xxxx \
GITHUB_REPOSITORY=dappnode/staker-test-util \
GITHUB_PR_NUMBER=123 \
docker-compose up --build
```

## TODO's

- [x] Implement a github adapter to interact with issues and PRs so we can automate report creation as well as the testing process.
  - Checks if snapshots exists for given clients and network
    - If they dont exist then write temporary file `download_in_progress` to signal other processes that snapshot download is in progress and:
      1. download snapshots
      2. extract snapshots
      3. mount snapshots to respective volumes 
      4. Write the block number of the snapshot downloaded to a file `snapshot_block_number` inside the respective volume
      5. remove temporary file `download_in_progress`
    - If they do exist then check:
      1. If file `download_in_progress` exists then exit
      2. If file `snapshot_block_number` exists then check if the latest snapshot available is newer, if so redownload snapshot as above
      3. If file `snapshot_block_number` does not exist then redownload snapshot as above
- test runner: executed once per test (manually or on PR) 
  - ensures environment before running tests
    - among other things it must check if the file `download_in_progress` exits, if so the test must wait until it is removed
  - runs tests
  - cleanup environment after tests

- [x] Implement a github adapter to interact with issues and PRs so we can automate report creation as well as the testing process.
- [x] Measure the time it takes every process in the test and add it to the report
- [x] Collect logs from containers and create a report
- [x] Add to the report the clients used
- [x] Implement timer check in the snapshot checker
- [ ] The test runner must remove the clients or unset the staker config on exit
- [x] Implement a composite adapter for the snapshot checker
- [x] Implement time tracker in the snapshot checker service to measure time taken for each action, specially download and extraction of snapshots
- [ ] Auto-updates for this dappnode must run much more often than production, so clients are always updated to latest versions
- [ ] Research how to release this SDK tool to be run from a github action directly
- [ ] Implement when manual trigger (`workflow_dispatch`) the clients will be passed as inputs, use them to create the staker config for the test
- [ ] Implement edit of `/usr/src/dappnode/DNCORE/docker-compose.yml` file to add env `TEST=true` and relaunch compose
- [ ] Print version of the clients 
    - [ ] For the EC it must be printed
- [ ] Add to the report the block of the snapshot used
- [ ] Consider always removing beacon volumes to ensure avoiding old states of chain and always start with the checkpoint sync
- [ ] Implement switch off of dappmanager cron that restarts containers of clients selected in stakers
- [ ] Silent the tar output when extracting snapshots or make it less verbose
- [ ] Consider adding a DB for reports and a UI to consume them
- [ ] Consider adding to report beaconcha validator url
- [ ] Consider adding support for multiple networks (mainnet, prater, etc.)
- [ ] Consider setting keystore and password through github secrets.
- [ ] How to ensure IPFS resilience, it must resolve production and dev IPFS hashes!
- [ ] Pass all the time measured in the ensurer and or runner and or cleaner to the test runner service for printing and report