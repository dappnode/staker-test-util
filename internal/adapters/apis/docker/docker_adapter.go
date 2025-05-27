package docker

import (
	"context"
	"errors"
	"fmt"

	"clients-test/internal/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
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

// StopAndGetVolumeTarget stops the container, checks for a single volume, and returns the volume target path
func (d *DockerAdapter) StopAndGetVolumeTarget(ctx context.Context, containerName string) (string, error) {
	logger.Debug("[DockerAdapter] StopAndGetVolumeTarget: inspecting container %s", containerName)
	containerJSON, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		logger.Error("[DockerAdapter] StopAndGetVolumeTarget: failed to inspect container: %v", err)
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	if len(containerJSON.Mounts) != 1 {
		logger.Error("[DockerAdapter] StopAndGetVolumeTarget: expected 1 volume, found %d", len(containerJSON.Mounts))
		return "", fmt.Errorf("expected exactly one volume for container %s, but found %d", containerName, len(containerJSON.Mounts))
	}
	mount := containerJSON.Mounts[0]
	if mount.Type != "volume" {
		logger.Error("[DockerAdapter] StopAndGetVolumeTarget: mount is not a docker volume, type=%s", mount.Type)
		return "", errors.New("the mount is not a docker volume")
	}
	volumeName := mount.Name
	volumeTarget := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)
	logger.Debug("[DockerAdapter] StopAndGetVolumeTarget: volumeName=%s volumeTarget=%s", volumeName, volumeTarget)

	// Stop the container
	logger.Debug("[DockerAdapter] StopAndGetVolumeTarget: stopping container %s", containerName)
	if err := d.cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		logger.Error("[DockerAdapter] StopAndGetVolumeTarget: failed to stop container: %v", err)
		return "", fmt.Errorf("failed to stop container: %w", err)
	}
	logger.Debug("[DockerAdapter] StopAndGetVolumeTarget: stopped container %s", containerName)

	return volumeTarget, nil
}

// StartContainer starts the given container
func (d *DockerAdapter) StartContainer(ctx context.Context, containerName string) error {
	logger.Debug("[DockerAdapter] StartContainer: starting container %s", containerName)
	if err := d.cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		logger.Error("[DockerAdapter] StartContainer: failed to start container: %v", err)
		return fmt.Errorf("failed to start container: %w", err)
	}
	logger.Debug("[DockerAdapter] StartContainer: started container %s", containerName)
	return nil
}
