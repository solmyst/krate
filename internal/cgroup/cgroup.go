package cgroup

import (
	"fmt"
	"os"
	"strconv"
)

func Setup(id string, memoryMB int64, cpuPct int) {
	path := "/sys/fs/cgroup/krate-" + id[:8]
	os.MkdirAll(path, 0755)

	memBytes := memoryMB * 1024 * 1024
	os.WriteFile(path+"/memory.max", []byte(strconv.FormatInt(memBytes, 10)), 0700)

	quota := cpuPct * 1000
	os.WriteFile(path+"/cpu.max", []byte(fmt.Sprintf("%d 100000", quota)), 0700)

	pid := strconv.Itoa(os.Getpid())
	os.WriteFile(path+"/cgroup.procs", []byte(pid), 0700)
}

func Cleanup(id string) {
	os.Remove("/sys/fs/cgroup/krate-" + id[:8])
}