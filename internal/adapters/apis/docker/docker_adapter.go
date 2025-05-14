package docker

import (
	"context"
	"errors"
	"fmt"

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
	containerJSON, err := d.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	if len(containerJSON.Mounts) != 1 {
		return "", fmt.Errorf("expected exactly one volume for container %s, but found %d", containerName, len(containerJSON.Mounts))
	}
	mount := containerJSON.Mounts[0]
	if mount.Type != "volume" {
		return "", errors.New("the mount is not a docker volume")
	}
	volumeName := mount.Name
	volumeTarget := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)

	// Stop the container
	if err := d.cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("failed to stop container: %w", err)
	}

	return volumeTarget, nil
}

// StartContainer starts the given container
func (d *DockerAdapter) StartContainer(ctx context.Context, containerName string) error {
	if err := d.cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}
