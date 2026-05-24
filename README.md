cat > ~/krate/README.md << 'EOF'
# krate 🐳

A container runtime built from scratch in Go. Implements core containerization primitives — Linux namespaces, cgroups v2, overlay filesystems — the same foundations Docker is built on.

## Features

- **Process isolation** via Linux namespaces (PID, UTS, Mount, Network)
- **Resource limiting** via cgroups v2 (memory + CPU)
- **Filesystem isolation** via chroot + overlay FS (copy-on-write layers)
- **Image management** — pull, cache, and list Alpine images
- **Container lifecycle** — run, stop, exec, ps, logs
- **HTTP daemon** — REST API to control containers programmatically
- **Web dashboard** — real-time container monitoring UI

## How it works

When you run `krate run alpine /bin/sh`:

1. Image is pulled and cached at `/var/lib/krate/images/`
2. Overlay FS is mounted — Alpine as lower (read-only), fresh upper layer for writes
3. Namespaces created via `clone()` syscall — `CLONE_NEWUTS | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWNET`
4. cgroups v2 enforce memory and CPU limits
5. chroot pivots into the overlay rootfs
6. `/proc`, `/tmp`, `/sys` mounted inside the container

## Usage

```bash
# pull an image
sudo ./krate pull alpine

# run a container
sudo ./krate run alpine /bin/sh

# with resource limits
sudo ./krate run --memory 50 --cpu 20 --name mybox alpine /bin/sh

# list running containers
sudo ./krate ps

# exec into a running container
sudo ./krate exec <id> /bin/sh

# stop a container
sudo ./krate stop <id>

# start HTTP daemon + dashboard
sudo ./krate daemon
# open http://localhost:6060
```

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /containers | List running containers |
| POST | /containers/run | Start a new container |
| DELETE | /containers/stop/:id | Stop a container |
| GET | /containers/logs/:id | Get container logs |
| GET | /images | List pulled images |

## Architecture
```text

krate/
├── cmd/              # CLI entrypoint (Cobra)
├── internal/
│   ├── container/    # run + child process logic
│   ├── cgroup/       # cgroup v2 resource limits
│   ├── image/        # image pull + cache
│   ├── overlay/      # overlay filesystem setup
│   ├── state/        # container state persistence
│   └── daemon/       # HTTP API server
└── web/              # dashboard UI

```

---

## Requirements

- Linux or WSL2
- Go 1.21+
- sudo (required for namespace + cgroup operations)

## Build

```bash
git clone https://github.com/solmyst/krate
cd krate
go build -o krate ./cmd/
sudo ./krate pull alpine
sudo ./krate run alpine /bin/sh
```
EOF

git add README.md
git commit -m "docs: add README"
git push
