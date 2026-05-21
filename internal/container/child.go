package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"krate/internal/cgroup"
)

func Child() {
	cfgJSON := os.Getenv("KRATE_CFG")
	if cfgJSON == "" {
		fmt.Fprintln(os.Stderr, "krate: missing container config")
		os.Exit(1)
	}

	var cfg childConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		fmt.Fprintln(os.Stderr, "krate: invalid config:", err)
		os.Exit(1)
	}

	// apply resource limits
	cgroup.Setup(cfg.ID, cfg.MemoryMB, cfg.CPUPct)
	defer cgroup.Cleanup(cfg.ID)

	// set container hostname
	syscall.Sethostname([]byte(cfg.Hostname))

	// bind /dev from host before chroot
	syscall.Mount("/dev", cfg.Rootfs+"/dev", "", syscall.MS_BIND|syscall.MS_REC, "")

	// chroot into overlay rootfs
	if err := syscall.Chroot(cfg.Rootfs); err != nil {
		fmt.Fprintln(os.Stderr, "chroot error:", err)
		os.Exit(1)
	}
	os.Chdir("/")

	// mount essential filesystems
	syscall.Mount("proc", "/proc", "proc", 0, "")
	syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, "")
	syscall.Mount("sysfs", "/sys", "sysfs", syscall.MS_RDONLY, "")

	// bring up loopback network
	exec.Command("ip", "link", "set", "lo", "up").Run()

	// run user command
	cmd := exec.Command(cfg.Cmd[0], cfg.Cmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}