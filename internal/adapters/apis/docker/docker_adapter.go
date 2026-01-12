package docker

import (
	"context"
	"fmt"

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
