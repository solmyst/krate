package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"krate/internal/container"
	"krate/internal/image"
	"krate/internal/state"
	"krate/internal/daemon"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "__child__" {
		container.Child()
		return
	}

	var memMB int64
	var cpuPct int
	var hostname string
	
daemonCmd := &cobra.Command{
    Use:   "daemon",
    Short: "Start the krate HTTP API daemon",
    RunE: func(cmd *cobra.Command, args []string) error {
        return daemon.Start("6060")
    },
}
	rootCmd := &cobra.Command{
		Use:   "krate",
		Short: "krate — a minimal container runtime",
	}

	// run
	runCmd := &cobra.Command{
		Use:   "run <image> <command> [args...]",
		Short: "Run a command in a new container",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return container.Run(container.Config{
				Image:    args[0],
				Cmd:      args[1:],
				MemoryMB: memMB,
				CPUPct:   cpuPct,
				Hostname: hostname,
			})
		},
	}
	runCmd.Flags().Int64VarP(&memMB, "memory", "m", 100, "Memory limit in MB")
	runCmd.Flags().IntVar(&cpuPct, "cpu", 50, "CPU limit (1-100%)")
	runCmd.Flags().StringVar(&hostname, "name", "krate-box", "Container hostname")

	// pull
	pullCmd := &cobra.Command{
		Use:   "pull <image>",
		Short: "Pull an image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return image.Pull(args[0])
		},
	}

	// images
	imagesCmd := &cobra.Command{
		Use:   "images",
		Short: "List pulled images",
		RunE: func(cmd *cobra.Command, args []string) error {
			imgs, err := image.List()
			if err != nil {
				return err
			}
			fmt.Printf("%-20s %-10s\n", "IMAGE", "SIZE")
			fmt.Println("------------------------------")
			for _, img := range imgs {
				fmt.Printf("%-20s %-10s\n", img.Name, img.Size)
			}
			return nil
		},
	}

	// ps
	psCmd := &cobra.Command{
		Use:   "ps",
		Short: "List running containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			containers, err := state.List()
			if err != nil {
				return err
			}
			fmt.Printf("%-14s %-12s %-20s %-10s\n", "CONTAINER ID", "IMAGE", "COMMAND", "UPTIME")
			fmt.Println(repeat("-", 60))
			for _, c := range containers {
				cmdStr := c.Cmd[0]
				uptime := time.Since(c.StartedAt).Round(time.Second)
				fmt.Printf("%-14s %-12s %-20s %-10s\n",
					c.ID[:12], c.Image, cmdStr, uptime)
			}
			if len(containers) == 0 {
				fmt.Println("No running containers")
			}
			return nil
		},
	}

	// stop
	stopCmd := &cobra.Command{
		Use:   "stop <container-id>",
		Short: "Stop a running container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := state.Get(args[0])
			if err != nil {
				return fmt.Errorf("container not found: %s", args[0])
			}
			proc, err := os.FindProcess(c.PID)
			if err != nil {
				return err
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return err
			}
			state.Delete(c.ID)
			fmt.Printf("Stopped container %s\n", c.ID[:12])
			return nil
		},
	}

	// logs
	logsCmd := &cobra.Command{
		Use:   "logs <container-id>",
		Short: "Show logs for a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := state.Get(args[0])
			if err != nil {
				return fmt.Errorf("container not found: %s", args[0])
			}
			data, err := os.ReadFile(c.LogFile)
			if err != nil {
				return fmt.Errorf("no logs found")
			}
			fmt.Print(string(data))
			return nil
		},
	}

	// exec
	execCmd := &cobra.Command{
		Use:   "exec <container-id> <command>",
		Short: "Run a command in a running container",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := state.Get(args[0])
			if err != nil {
				return fmt.Errorf("container not found: %s", args[0])
			}
			pid := strconv.Itoa(c.PID)
			nsArgs := []string{
				"-t", pid,
				"--uts", "--pid", "--mount", "--net",
				"--",
			}
			nsArgs = append(nsArgs, args[1:]...)
			execCmd := exec.Command("nsenter", nsArgs...)
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			return execCmd.Run()
		},
	}

	rootCmd.AddCommand(runCmd, pullCmd, imagesCmd, psCmd, stopCmd, logsCmd, execCmd, daemonCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}