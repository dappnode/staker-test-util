package mount

import (
	"context"
	"fmt"
	"os/exec"
)

type MountAdapter struct{}

func NewMountAdapter() *MountAdapter {
	return &MountAdapter{}
}

// MountNFS mounts the NFS share at srcPath to the targetPath using sudo mount -t nfs
func (m *MountAdapter) MountNFS(ctx context.Context, srcPath, targetPath string) error {
	mountCmd := exec.CommandContext(ctx, "sudo", "mount", "-t", "nfs", srcPath, targetPath)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount %s onto %s: %v\n%s", srcPath, targetPath, err, string(output))
	}
	return nil
}
