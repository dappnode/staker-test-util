package mount

import (
	"context"
	"fmt"
	"os/exec"

	"clients-test/internal/logger"
)

type MountAdapter struct {
	logPrefix string
}

func NewMountAdapter() *MountAdapter {
	return &MountAdapter{logPrefix: "MountAdapter"}
}

// MountNFS mounts the NFS share at srcPath to the targetPath using sudo mount -t nfs
func (m *MountAdapter) MountNFS(ctx context.Context, srcPath, targetPath string) error {
	logger.DebugWithPrefix(m.logPrefix, "MountNFS called with srcPath=%s, targetPath=%s", srcPath, targetPath)
	mountCmd := exec.CommandContext(ctx, "sudo", "mount", "-t", "nfs", srcPath, targetPath)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		logger.ErrorWithPrefix(m.logPrefix, "Failed to mount %s onto %s: %v\n%s", srcPath, targetPath, err, string(output))
		return fmt.Errorf("failed to mount %s onto %s: %v\n%s", srcPath, targetPath, err, string(output))
	}
	logger.DebugWithPrefix(m.logPrefix, "Successfully mounted %s onto %s", srcPath, targetPath)
	return nil
}

// UnmountNFS unmounts the filesystem at targetPath using sudo umount
func (m *MountAdapter) UnmountNFS(ctx context.Context, targetPath string) error {
	logger.DebugWithPrefix(m.logPrefix, "UnmountNFS called with targetPath=%s", targetPath)
	umountCmd := exec.CommandContext(ctx, "sudo", "umount", targetPath)
	output, err := umountCmd.CombinedOutput()
	if err != nil {
		logger.ErrorWithPrefix(m.logPrefix, "Failed to unmount %s: %v\n%s", targetPath, err, string(output))
		return fmt.Errorf("failed to unmount %s: %v\n%s", targetPath, err, string(output))
	}
	logger.DebugWithPrefix(m.logPrefix, "Successfully unmounted %s", targetPath)
	return nil
}
