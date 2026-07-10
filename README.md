# Krate 🐳

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/solmyst/krate/go.yml?branch=main&style=flat-square)](https://github.com/solmyst/krate/actions)

Krate is a minimal, zero-dependency container runtime built from scratch in Go, implementing containerization from first principles using direct Linux kernel namespace isolation, cgroups v2 resource limits, and copy-on-write overlay filesystems.

---

## Architecture

Krate integrates core Linux kernel virtualization primitives into a unified, low-overhead container lifecycle wrapper.

```mermaid
graph TD
    A[krate run CLI] -->|1. Resolve Image| B(Pull Alpine tar.gz & Extract to templates)
    A -->|2. Mount Storage| C(Mount OverlayFS: read-only template lower + writable upper/work)
    A -->|3. Fork Process| D[Fork /proc/self/exe as __child__ with Cloneflags]
    D -->|CLONE_NEWUTS | NEWPID | NEWNS | NEWNET| E[Child Namespace Process]
    E -->|4. Limit Resources| F(Write cgroups v2 cpu.max & memory.max)
    E -->|5. Bind Dev & Chroot| G(Bind mount host /dev, syscall.Chroot, syscall.Chdir)
    E -->|6. Mount Core FS| H(Mount proc, sysfs, tmpfs inside jail)
    E -->|7. Network Setup| I(Bring loopback 'lo' interface up)
    E -->|8. Execute Binary| J[Exec User Command e.g., /bin/sh]
```

### Virtualization Primitives
- **Process Isolation (Linux Namespaces)**: Uses `clone` flags (`CLONE_NEWUTS`, `CLONE_NEWPID`, `CLONE_NEWNS`, `CLONE_NEWNET`) to give the container process its own hostname, process table, mount tables, and network interfaces.
- **Resource Control (cgroups v2)**: Restricts physical system resource access by registering the container process's PID with `cgroup.procs` inside a dedicated controller slice. Enforces maximum memory footprint via `memory.max` and limits CPU usage using fractional quota periods via `cpu.max`.
- **Filesystem Layering (OverlayFS)**: Mounts an ephemeral overlay filesystem combining a read-only distribution image (the lower directory) and a container-specific writable overlay (upper and work directories) into a unified rootfs folder.

---

## Features

- **Namespace Isolation**: Full PID, UTS (hostname), Mount (isolated rootfs), and Network (loopback) isolation.
- **Resource Limits**: Hardware constraints on memory (MB) and CPU usage (1-100% quota limits) using unified cgroups v2.
- **Copy-on-Write Storage**: High-performance, isolated storage layers using standard Linux OverlayFS.
- **Secure Chroot**: Strict filesystem boundary isolation with clean, runtime-specific mounts for `/proc`, `/sys`, and `/tmp`.
- **Image Lifecycle**: Seamless download and extraction of minimal Alpine Linux distributions directly from official mirrors.
- **State Management**: Automatic container status, uptime, configuration, and logging persistence inside `/var/lib/krate`.
- **Control Daemon & REST API**: HTTP service supporting remote programmatic control over images and container runs/stops.
- **Web Console Dashboard**: A built-in web frontend serving real-time statistics and terminal output logs.

---

## Project Structure

| Directory | Description |
| :--- | :--- |
| [`cmd/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/cmd) | Command line interface utilizing Cobra. Handles configuration, commands, and child re-execution (`__child__`) setup. |
| [`internal/cgroup/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/cgroup) | Interacts with the cgroups v2 interface `/sys/fs/cgroup` to apply hard resource limits. |
| [`internal/container/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/container) | Orchestrates container execution flow, fork-re-exec patterns, namespace configuration, mounts, and syscall execution. |
| [`internal/daemon/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/daemon) | Embedded HTTP server implementing the REST API and serving the container dashboard. |
| [`internal/image/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/image) | Handles remote downloading, verification, caching, and extraction of Alpine minirootfs images. |
| [`internal/overlay/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/overlay) | Provisions and unmounts OverlayFS paths (`lowerdir`, `upperdir`, `workdir`, `merged`). |
| [`internal/state/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/internal/state) | Manages container status database, active PIDs, logs, and uptime records. |
| [`web/`](file:///c:/Users/Anush%20Gupta/Documents/GitHub/krate/web) | Real-time container telemetry and management UI. |

---

## How It Works

Krate uses a multi-phase system flow to instantiate isolated container environments securely:

1. **Parent Re-Execution**: Go's runtime scheduler does not play well with namespace cloning in multi-threaded programs. Krate solves this by using a re-execution trick. When you run `krate run`, the parent process initializes the overlay directory mounts, saves container state metadata, and executes `/proc/self/exe` with a custom argument `__child__`. The `syscall.SysProcAttr` is configured with `Cloneflags` to initialize new UTS, PID, Mount, and Network namespaces.
2. **Namespace Birth**: The newly spawned child process starts execution. Because it is in new namespaces, it is PID `1` within its own PID tree, cannot see host mount tables, and starts with an isolated network namespace containing only an unconfigured loopback device.
3. **Resource Attachment**: The child process reads configuration variables passed through the environment, creates a cgroup slice under `/sys/fs/cgroup/krate-<container_id>`, configures memory and CPU limits, and writes its own PID to `cgroup.procs`.
4. **Filesystem Pivot**: To secure filesystem access, the child binds the host's `/dev` nodes to `/dev` inside the merged overlayfs folder, executes `syscall.Chroot` to lock itself within the overlay root, changes the directory to `/`, and mounts new isolated instances of `proc` to `/proc`, `sysfs` to `/sys` (read-only), and `tmpfs` to `/tmp`.
5. **Initialization**: The child brings up the loopback interface (`lo`) via an `ip link` syscall wrapper and finally invokes the user-specified binary (e.g. `/bin/sh`) inside the isolated, resource-constrained container.

---

## Requirements

- **Linux Kernel**: A modern Linux system supporting namespaces, cgroups v2 unified hierarchy, and OverlayFS. Alternatively, WSL2 running a modern kernel is supported.
- **Go**: Version 1.21 or higher.
- **Privileges**: Administrative privileges (`sudo` or root) are required to execute namespaces manipulation, mounting, and cgroup management.

---

## Installation & Build

Compile the project from source:

```bash
# Clone the repository
git clone https://github.com/solmyst/krate.git
cd krate

# Compile the binary
go build -o krate ./cmd/

# Copy the web dashboard assets to the local configuration directory
sudo mkdir -p /var/lib/krate/web
sudo cp web/index.html /var/lib/krate/web/
```

---

## CLI Usage

### Pulling an Image
Pull the standard Alpine Linux root filesystem:
```bash
sudo ./krate pull alpine
```

### Running a Container
Launch an interactive shell inside a container with default parameters:
```bash
sudo ./krate run alpine /bin/sh
```

### Running with Resource Limits
Run a container named `db-node` with memory capped at 50MB and CPU execution quota limited to 20%:
```bash
sudo ./krate run --memory 50 --cpu 20 --name db-node alpine /bin/sh
```

### Executing Commands in a Running Container
Enter a running container's namespaces to execute debugging commands:
```bash
sudo ./krate exec <container-id> /bin/sh
```

### Managing Containers
```bash
# List active containers
sudo ./krate ps

# View stdout/stderr console logs of a container
sudo ./krate logs <container-id>

# Stop a container by sending SIGTERM to its process group
sudo ./krate stop <container-id>
```

### Starting the Daemon & Web Dashboard
Start the HTTP daemon to control your containers programmatically:
```bash
sudo ./krate daemon
```
Access the real-time container visualization dashboard at **http://localhost:6060**.

---

## REST API

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/containers` | Lists running containers. |
| `POST` | `/containers/run` | Spawns a container in the background. |
| `DELETE` | `/containers/stop/:id` | Terminates a running container. |
| `GET` | `/containers/logs/:id` | Returns stdout and stderr logs. |
| `GET` | `/images` | Lists pulled minirootfs images. |

---

## Why I Built This

I built Krate to demystify the magic of container runtimes like Docker and containerd by implementing their core virtualization mechanics myself. By building process isolation, filesystem overlaying, and resource limiting directly using Linux system calls and cgroups v2, I gained a deep, hands-on understanding of containerization at the kernel level.

---

## Contributing

1. Fork this repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'feat: add support for bridging networks'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
