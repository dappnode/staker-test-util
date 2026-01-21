package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"clients-test/internal/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerAdapter struct {
	cli       *client.Client
	logPrefix string
}

func NewDockerAdapter() (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerAdapter{cli: cli, logPrefix: "DockerAdapter"}, nil
}

// StopAndGetVolumeTarget stops the container, checks for a single volume, and returns the volume target path
func (d *DockerAdapter) StopAndGetVolumeTarget(ctx context.Context, containerName string, containerVolumeName string) (string, error) {
	logger.DebugWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: inspecting container %s", containerName)
	containerJSON, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: failed to inspect container: %v", err)
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	// the volumeName is the mount one which name is equal to the containerVolumeName, find it
	// find the mount which name is equal to containerVolumeName

	for _, mount := range containerJSON.Mounts {
		if mount.Name == containerVolumeName {
			volumeName := mount.Name

			volumeTarget := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)
			logger.DebugWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: volumeName=%s volumeTarget=%s", volumeName, volumeTarget)

			// Stop the container
			logger.DebugWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: stopping container %s", containerName)
			if err := d.cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
				logger.ErrorWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: failed to stop container: %v", err)
				return "", fmt.Errorf("failed to stop container: %w", err)
			}
			logger.DebugWithPrefix(d.logPrefix, "StopAndGetVolumeTarget: stopped container %s", containerName)

			return volumeTarget, nil
		}
	}

	return "", fmt.Errorf("failed to find volume mount %s in container %s", containerVolumeName, containerName)
}

// StartContainer starts the given container
func (d *DockerAdapter) StartContainer(ctx context.Context, containerName string) error {
	logger.DebugWithPrefix(d.logPrefix, "StartContainer: starting container %s", containerName)
	if err := d.cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "StartContainer: failed to start container: %v", err)
		return fmt.Errorf("failed to start container: %w", err)
	}
	logger.DebugWithPrefix(d.logPrefix, "StartContainer: started container %s", containerName)
	return nil
}

// GetContainerErrorLogs retrieves error logs from a container within a time range
// Returns up to maxLines error lines (lines containing "error", "err", "fatal", "panic", etc.)
func (d *DockerAdapter) GetContainerErrorLogs(ctx context.Context, containerName string, since, until time.Time, maxLines int) ([]string, error) {
	logger.DebugWithPrefix(d.logPrefix, "GetContainerErrorLogs: fetching logs for container %s from %v to %v", containerName, since, until)

	// Check if container exists first
	_, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		logger.WarnWithPrefix(d.logPrefix, "GetContainerErrorLogs: container %s not found or not accessible: %v", containerName, err)
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
		logger.WarnWithPrefix(d.logPrefix, "GetContainerErrorLogs: failed to get logs for %s: %v", containerName, err)
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
					logger.DebugWithPrefix(d.logPrefix, "GetContainerErrorLogs: found %d error lines for %s (max reached)", len(errorLines), containerName)
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

	logger.DebugWithPrefix(d.logPrefix, "GetContainerErrorLogs: found %d error lines for %s", len(errorLines), containerName)
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
			logger.WarnWithPrefix(d.logPrefix, "CollectAllContainerErrorLogs: error collecting logs for %s: %v", name, err)
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
	logger.DebugWithPrefix(d.logPrefix, "StopContainer: stopping container %s", containerName)

	// Check if container exists first
	_, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		logger.WarnWithPrefix(d.logPrefix, "StopContainer: container %s not found or not accessible: %v", containerName, err)
		return nil // Return nil - container might not exist
	}

	if err := d.cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		logger.ErrorWithPrefix(d.logPrefix, "StopContainer: failed to stop container: %v", err)
		return fmt.Errorf("failed to stop container: %w", err)
	}

	logger.DebugWithPrefix(d.logPrefix, "StopContainer: stopped container %s", containerName)
	return nil
}
