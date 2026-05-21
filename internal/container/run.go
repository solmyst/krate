package container

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"krate/internal/image"
	"krate/internal/overlay"
	"krate/internal/state"
)

const krateBase = "/var/lib/krate"

type Config struct {
	Image    string
	Cmd      []string
	MemoryMB int64
	CPUPct   int
	Hostname string
}

type childConfig struct {
	Rootfs   string   `json:"rootfs"`
	Cmd      []string `json:"cmd"`
	MemoryMB int64    `json:"memory_mb"`
	CPUPct   int      `json:"cpu_pct"`
	Hostname string   `json:"hostname"`
	ID       string   `json:"id"`
	LogFile  string   `json:"log_file"`
}

func Run(cfg Config) error {
	imgPath, err := image.Ensure(cfg.Image)
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}

	id := generateID()
	containerDir := filepath.Join(krateBase, "containers", id)
	logFile := filepath.Join(krateBase, "logs", id+".log")

	os.MkdirAll(filepath.Join(krateBase, "logs"), 0755)

	rootfs, err := overlay.Setup(imgPath, containerDir)
	if err != nil {
		return fmt.Errorf("overlay: %w", err)
	}
	defer overlay.Cleanup(containerDir)

	cc := childConfig{
		Rootfs:   rootfs,
		Cmd:      cfg.Cmd,
		MemoryMB: cfg.MemoryMB,
		CPUPct:   cfg.CPUPct,
		Hostname: cfg.Hostname,
		ID:       id,
		LogFile:  logFile,
	}
	cfgJSON, _ := json.Marshal(cc)

	fmt.Printf("\nContainer ID : %s\n", id[:12])
	fmt.Printf("Image        : %s\n", cfg.Image)
	fmt.Printf("Memory       : %dMB\n", cfg.MemoryMB)
	fmt.Printf("CPU          : %d%%\n", cfg.CPUPct)
	fmt.Printf("Hostname     : %s\n\n", cfg.Hostname)

	cmd := exec.Command("/proc/self/exe", "__child__")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"KRATE_CFG=" + string(cfgJSON)}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// save container state
	state.Save(state.Container{
		ID:        id,
		Image:     cfg.Image,
		Cmd:       cfg.Cmd,
		Hostname:  cfg.Hostname,
		PID:       cmd.Process.Pid,
		StartedAt: time.Now(),
		LogFile:   logFile,
		RootFS:    rootfs,
	})

	err = cmd.Wait()
	state.Delete(id)
	return err
}

func generateID() string {
	rand.Seed(time.Now().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}