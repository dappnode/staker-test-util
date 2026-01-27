package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerAdapter struct {
	cli *client.Client
}

func NewDockerAdapter() (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerAdapter{cli: cli}, nil
}

// StartContainer starts the given container
func (d *DockerAdapter) StartContainer(ctx context.Context, containerName string) error {
	if err := d.cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// GetContainerErrorLogs retrieves error logs from a container within a time range
// Returns up to maxLines error lines (lines containing "error", "err", "fatal", "panic", etc.)
func (d *DockerAdapter) GetContainerErrorLogs(ctx context.Context, containerName string, since, until time.Time, maxLines int) ([]string, error) {
	// Check if container exists first
	_, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, nil // Return nil instead of error - container might not exist yet
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since.Format(time.RFC3339),
		Until:      until.Format(time.RFC3339),
		Timestamps: true,
	}

	reader, err := d.cli.ContainerLogs(ctx, containerName, options)
	if err != nil {
		return nil, nil // Return nil instead of error - might be transient
	}
	defer reader.Close()

	errorLines := make([]string, 0)
	errorKeywords := []string{"error", "err:", "fatal", "panic", "exception", "failed", "failure"}

	// Docker logs have an 8-byte header for multiplexed streams
	// We need to handle this properly
	buf := make([]byte, 8)
	for {
		// Read the header
		_, err := io.ReadFull(reader, buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			// Try reading as plain text if header read fails
			break
		}

		// Get the size of the log line from header bytes 4-7
		size := int(buf[4])<<24 | int(buf[5])<<16 | int(buf[6])<<8 | int(buf[7])

		// Read the actual log line
		line := make([]byte, size)
		_, err = io.ReadFull(reader, line)
		if err != nil {
			break
		}

		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" {
			continue
		}

		// Check if line contains error keywords (case-insensitive)
		lineLower := strings.ToLower(lineStr)
		for _, keyword := range errorKeywords {
			if strings.Contains(lineLower, keyword) {
				// Truncate very long lines
				if len(lineStr) > 500 {
					lineStr = lineStr[:500] + "..."
				}
				errorLines = append(errorLines, lineStr)
				if len(errorLines) >= maxLines {
					return errorLines, nil
				}
				break
			}
		}
	}

	// If the multiplexed read failed, try reading as plain text
	if len(errorLines) == 0 {
		reader2, err := d.cli.ContainerLogs(ctx, containerName, options)
		if err != nil {
			return nil, nil
		}
		defer reader2.Close()

		scanner := bufio.NewScanner(reader2)
		for scanner.Scan() {
			lineStr := strings.TrimSpace(scanner.Text())
			if lineStr == "" {
				continue
			}

			lineLower := strings.ToLower(lineStr)
			for _, keyword := range errorKeywords {
				if strings.Contains(lineLower, keyword) {
					if len(lineStr) > 500 {
						lineStr = lineStr[:500] + "..."
					}
					errorLines = append(errorLines, lineStr)
					if len(errorLines) >= maxLines {
						return errorLines, nil
					}
					break
				}
			}
		}
	}

	return errorLines, nil
}

// CollectAllContainerErrorLogs collects error logs from multiple containers
func (d *DockerAdapter) CollectAllContainerErrorLogs(ctx context.Context, containerNames []string, since, until time.Time, maxLinesPerContainer int) map[string][]string {
	result := make(map[string][]string)

	for _, name := range containerNames {
		if name == "" {
			continue
		}
		lines, err := d.GetContainerErrorLogs(ctx, name, since, until, maxLinesPerContainer)
		if err != nil {
			continue
		}
		if len(lines) > 0 {
			result[name] = lines
		}
	}

	return result
}

// StopContainer stops a container by name
func (d *DockerAdapter) StopContainer(ctx context.Context, containerName string) error {
	// Check if container exists first
	_, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil // Return nil - container might not exist
	}

	if err := d.cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// ListContainersByPrefix returns a list of container IDs that match the given name prefix
func (d *DockerAdapter) ListContainersByPrefix(ctx context.Context, prefix string) ([]string, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", prefix)

	containers, err := d.cli.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var ids []string
	for _, c := range containers {
		ids = append(ids, c.ID)
	}

	return ids, nil
}

// StopContainerWithTimeout stops a container by ID with a specific timeout in seconds
func (d *DockerAdapter) StopContainerWithTimeout(ctx context.Context, containerID string, timeoutSeconds int) error {
	timeout := timeoutSeconds
	if err := d.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

const (
	alpineImage = "alpine:latest"
)

// RunSnapshotDownload runs an Alpine container to download and extract a snapshot
// Uses aria2c for parallel HTTP range request downloads and zstd for decompression
// Returns the duration of the download operation
func (d *DockerAdapter) RunSnapshotDownload(ctx context.Context, containerName, clientName, network, targetPath, baseURL string) error {
	// Pull alpine image if not present
	if err := d.ensureImage(ctx, alpineImage); err != nil {
		return fmt.Errorf("failed to ensure alpine image: %w", err)
	}

	// Build the shell script for download and extraction
	// Optimizations:
	// - aria2c -x16 -s16: 16 parallel connections using HTTP range requests
	// - zstd -T0: multi-threaded decompression using all CPU cores
	// - No -v flag on tar: avoid printing every filename (major speedup)
	shellScript := fmt.Sprintf(`
set -e
apk add --no-cache aria2 tar zstd pv bash curl > /dev/null 2>&1
BLOCK_NUMBER=$(curl -sf %s/%s/%s/latest)
SNAPSHOT_URL="%s/%s/%s/${BLOCK_NUMBER}/snapshot.tar.zst"
echo "[%s] Downloading snapshot for block number: ${BLOCK_NUMBER}"
echo "[%s] Using 16 parallel connections with HTTP range requests"
aria2c -x16 -s16 --file-allocation=none --console-log-level=warn --summary-interval=30 --show-console-readout=false -d /data -o snapshot.tar.zst "${SNAPSHOT_URL}" 2>&1 | awk '/^\[#/{print "[%s] " $0; fflush()}'
echo "[%s] Download complete. Extracting with $(nproc) CPU cores..."
bash -c 'pv -f -i 30 -N "%s" -ptebar /data/snapshot.tar.zst 2> >(tr "\r" "\n" >&2) | zstd -d -T0 | tar -xf - -C /data'
rm -f /data/snapshot.tar.zst
echo "[%s] Snapshot extraction complete"
`, baseURL, network, clientName, baseURL, network, clientName, clientName, clientName, clientName, clientName, clientName, clientName)

	// Create container config
	config := &container.Config{
		Image:      alpineImage,
		Entrypoint: []string{"/bin/sh"},
		Cmd:        []string{"-c", shellScript},
		Tty:        false,
	}

	// Create host config with volume mount
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: targetPath,
				Target: "/data",
			},
		},
		AutoRemove: true,
	}

	// Create the container
	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Attach to container output for real-time logging
	attachResp, err := d.cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err == nil {
		defer attachResp.Close()
		// Copy output to stdout/stderr in a goroutine
		go func() {
			_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
		}()
	}

	// Wait for container to finish
	statusCh, errCh := d.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("container exited with status %d", status.StatusCode)
		}
	}

	return nil
}

// ensureImage pulls an image if it's not already present locally
func (d *DockerAdapter) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, err := d.cli.ImageInspect(ctx, imageName)
	if err == nil {
		return nil // Image already exists
	}

	reader, err := d.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Discard the output but wait for pull to complete
	_, _ = io.Copy(io.Discard, reader)

	return nil
}

// RemoveVolumeData removes all data from a Docker volume by deleting its contents
func (d *DockerAdapter) RemoveVolumeData(ctx context.Context, volumeTargetPath string) error {
	// remove all files in the volume target path
	dirEntries, err := os.ReadDir(volumeTargetPath)
	if err != nil {
		return fmt.Errorf("failed to read volume directory: %w", err)
	}

	for _, entry := range dirEntries {
		entryPath := fmt.Sprintf("%s/%s", volumeTargetPath, entry.Name())
		if entry.IsDir() {
			if err := os.RemoveAll(entryPath); err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", entryPath, err)
			}
		} else {
			if err := os.Remove(entryPath); err != nil {
				return fmt.Errorf("failed to remove file %s: %w", entryPath, err)
			}
		}
	}

	return nil
}
