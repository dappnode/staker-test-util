package mount

import (
	"context"
	"fmt"
	"os/exec"

	"clients-test/internal/logger"
)

type MountAdapter struct{}

func NewMountAdapter() *MountAdapter {
	return &MountAdapter{}
}

// MountNFS mounts the NFS share at srcPath to the targetPath using sudo mount -t nfs
func (m *MountAdapter) MountNFS(ctx context.Context, srcPath, targetPath string) error {
	logger.Debug("MountNFS called with srcPath=%s, targetPath=%s", srcPath, targetPath)
	mountCmd := exec.CommandContext(ctx, "sudo", "mount", "-t", "nfs", srcPath, targetPath)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to mount %s onto %s: %v\n%s", srcPath, targetPath, err, string(output))
		return fmt.Errorf("failed to mount %s onto %s: %v\n%s", srcPath, targetPath, err, string(output))
	}
	logger.Debug("Successfully mounted %s onto %s", srcPath, targetPath)
	return nil
}

// UnmountNFS unmounts the filesystem at targetPath using sudo umount
func (m *MountAdapter) UnmountNFS(ctx context.Context, targetPath string) error {
	logger.Debug("UnmountNFS called with targetPath=%s", targetPath)
	umountCmd := exec.CommandContext(ctx, "sudo", "umount", targetPath)
	output, err := umountCmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to unmount %s: %v\n%s", targetPath, err, string(output))
		return fmt.Errorf("failed to unmount %s: %v\n%s", targetPath, err, string(output))
	}
	logger.Debug("Successfully unmounted %s", targetPath)
	return nil
}
