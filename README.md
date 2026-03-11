# Serverless CLI

A CLI that runs your Python code on your Kubernetes cluster without managing infrastructure. Submit one-off jobs, async jobs, cron jobs, or long-running services from local source code—no Docker build or image push required.

## Features

- **One-off jobs** – Run a Python script once and stream logs until it completes
- **Async jobs** – Submit a job and return immediately; check status with `list` and `logs`
- **Cron jobs** – Schedule Python code with a cron expression (e.g. `0 * * * *` for hourly)
- **Services** – Run a Python app (e.g. Flask, FastAPI) as a Deployment with a NodePort Service

Source code is packaged into a Kubernetes ConfigMap and executed in a runner container (`matansalto/serverless-python`) that mounts the code at `/opt/code`.

---

## Prerequisites

- **Go 1.21+** (for building from source)
- **Kubernetes cluster** with `kubectl` access
- **kubeconfig** – `KUBECONFIG` set or `~/.kube/config` present so the CLI can talk to your cluster
- **Python workloads** – Your code runs in the runner image; use the standard library or ensure dependencies are available in that image

---

## Installation

### From source

```bash
git clone https://github.com/your-org/Serverless-CLI.git
cd Serverless-CLI
go build -o serverless-cli .
# Optional: move to PATH
sudo mv serverless-cli /usr/local/bin/
```

### Verify

```bash
serverless-cli --help
serverless-cli run --help
```

---

## Usage

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | (none) | Path to kubeconfig file. If unset, uses `$KUBECONFIG` or `~/.kube/config`. |
| `--namespace` | `serverless-workloads` | Kubernetes namespace for workloads. |

Example:

```bash
serverless-cli --namespace my-namespace list
```

### Commands

#### `run one-off <source-path> [args...]`

Run a Python program once. Creates a Job, streams logs until it completes.

```bash
# Directory with main.py
serverless-cli run one-off ./my-script

# Single file
serverless-cli run one-off script.py

# Custom entrypoint and name
serverless-cli run one-off ./app --entrypoint run.py --name my-job
```

| Flag | Default | Description |
|------|---------|-------------|
| `--entrypoint` | `main.py` (dir) or filename (file) | Script to run under `/opt/code`. |
| `--name` | Generated (e.g. `slp-my-script-abc123`) | Job name. |

---

#### `run async <source-path> [args...]`

Run a Python program asynchronously. Creates a Job and returns immediately (no log streaming).

```bash
serverless-cli run async ./my-script
# Check status: serverless-cli list
# View logs:   serverless-cli logs <job-name>
```

Same flags as `run one-off`: `--entrypoint`, `--name`.

---

#### `run cron <source-path> [args...]`

Create a CronJob that runs your Python code on a schedule.

```bash
serverless-cli run cron ./daily-report --schedule "0 9 * * *"   # Every day at 09:00
serverless-cli run cron ./hourly.py --schedule "0 * * * *"      # Every hour
serverless-cli run cron ./task --schedule "*/5 * * * *" --name my-cron
```

| Flag | Default | Description |
|------|---------|-------------|
| `--entrypoint` | `main.py` (dir) or filename (file) | Script to run. |
| `--name` | Generated | CronJob name. |
| `--schedule` | **(required)** | Cron schedule (e.g. `0 * * * *`). |

---

#### `run service <source-path> [args...]`

Run a Python program as a long-running service (Deployment + NodePort Service). Your app should bind to the port (e.g. Flask on `0.0.0.0:PORT`). The CLI prints the NodePort and a suggested URL.

```bash
serverless-cli run service ./webapp
serverless-cli run service ./api --port 8000 --name my-api
```

| Flag | Default | Description |
|------|---------|-------------|
| `--entrypoint` | `main.py` (dir) or filename (file) | Script to run. |
| `--name` | Generated | Deployment/Service name. |
| `--port` | `8080` | Container port the app listens on. |

Example app (bind to `PORT` from env or a fixed port):

```python
# main.py
import os
import http.server
import socketserver

PORT = int(os.environ.get("PORT", 8080))
with socketserver.TCPServer(("", PORT), http.server.SimpleHTTPRequestHandler) as httpd:
    httpd.serve_forever()
```

---

#### `list`

List all serverless workloads (Jobs, CronJobs, Deployments) in the configured namespace.

```bash
serverless-cli list
```

Output columns: NAME, TYPE, STATUS, AGE, SCHEDULE, URL (for services).

---

#### `logs <workload-name>`

Stream logs for a workload by name.

- **Job** – Streams until the job completes.
- **Deployment (service)** – Streams logs from one of its pods.
- **CronJob** – Streams logs from the most recent Job created by that CronJob.

```bash
serverless-cli logs slp-my-script-abc123
serverless-cli logs slp-webapp-xyz789
```

---

#### `delete <workload-name>`

Delete a workload (Job, CronJob, or Deployment/Service) and its source ConfigMap.

```bash
serverless-cli delete slp-my-script-abc123
```

---

## Limits and behavior

- **Source size** – Total packaged source must be ≤ **1 MiB** (ConfigMap limit). Use a single file or a small directory.
- **Entrypoint** – For a **directory**, default is `main.py`. For a **single file**, default is that file’s name. Override with `--entrypoint`.
- **Runner image** – Workloads use `matansalto/serverless-python:1.0.0`. Code is mounted at `/opt/code`; `SLP_ENTRYPOINT` is set to the entrypoint script.
- **Namespace** – All resources are created in the namespace set by `--namespace` (default: `serverless-workloads`). Create it if needed: `kubectl create namespace serverless-workloads`.

---

## Developer guide

### Project structure

```
.
├── main.go              # Entrypoint; calls cmd.Execute()
├── cmd/
│   ├── root.go          # Root command, global flags (kubeconfig, namespace)
│   ├── run/
│   │   ├── run.go       # run command group
│   │   ├── one-off.go   # run one-off
│   │   ├── async.go     # run async
│   │   ├── cron.go      # run cron
│   │   └── service.go   # run service
│   ├── list/list.go     # list workloads
│   ├── logs/logs.go     # logs <name>
│   └── delete/delete.go # delete <name>
├── pkg/
│   ├── kube/            # Kubernetes API: Jobs, CronJobs, Deployments, Services, ConfigMaps, logs
│   ├── packager/        # Build file map from path, convert to ConfigMap data/volume items
│   ├── runner/          # High-level RunSource, RunCronSource, RunServiceSource, CleanupSource
│   └── utils/           # Helpers (e.g. colored log writer)
```

### Build and test

```bash
# Build
go build -o serverless-cli .

# Run tests
go test ./...

# Test a specific package
go test ./pkg/runner/...
go test ./pkg/kube/...
go test ./pkg/packager/...
```

### Adding a new command

1. Create a new file under `cmd/` (or a subpackage like `cmd/list/`).
2. Define a `cobra.Command` and implement `RunE`.
3. Register it in `cmd/root.go` with `rootCmd.AddCommand(...)`.
4. Use `cmd.Root().PersistentFlags().GetString("namespace")` (and optionally `kubeconfig`) for global settings.
5. Use `kube.NewClientSet()` for cluster access (respects `KUBECONFIG` / default kubeconfig).

### Dependencies

- [spf13/cobra](https://github.com/spf13/cobra) – CLI framework
- [k8s.io/client-go](https://github.com/kubernetes/client-go) – Kubernetes API client

Managed via Go modules; run `go mod tidy` after changing imports.

---
