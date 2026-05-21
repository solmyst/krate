package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func Setup(imgPath, containerDir string) (string, error) {
	lower := imgPath
	upper := filepath.Join(containerDir, "upper")
	work := filepath.Join(containerDir, "work")
	merged := filepath.Join(containerDir, "merged")

	for _, dir := range []string{upper, work, merged} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
	if err := syscall.Mount("overlay", merged, "overlay", 0, opts); err != nil {
		return "", fmt.Errorf("mount overlay: %w", err)
	}

	return merged, nil
}

func Cleanup(containerDir string) {
	merged := filepath.Join(containerDir, "merged")
	syscall.Unmount(merged, syscall.MNT_DETACH)
	os.RemoveAll(containerDir)
}