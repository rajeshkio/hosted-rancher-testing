# rancher-version-test

A Go tool that provisions a downstream Kubernetes cluster via Rancher, runs basic health checks, and optionally tests a Kubernetes version upgrade. Uses Terraform to manage cloud infrastructure.

## What it does

1. Provisions a downstream k3s cluster on a cloud provider via Rancher
2. Deploys a test nginx application
3. Verifies pod health, logs, and exec
4. Optionally upgrades the cluster to a newer k3s version and re-validates

If a run fails midway, it resumes from where it left off using a local state file.

## Requirements

- Go 1.21+
- Terraform
- kubectl
- A running Rancher instance
- Cloud provider account (DigitalOcean supported, AWS/Azure planned)

## Setup

Copy the example env file and fill in your values:

```
cp .env.example .env
```

```
RANCHER_URL=https://rancher.example.com
RANCHER_TOKEN=token-xxxxx:yyyyyyy
RANCHER_VERSION=2.9.0
K3S_VERSION=v1.33.8+k3s1
K3S_UPGRADE_VERSION=v1.34.5+k3s1   # optional, leave empty to skip upgrade test
CLOUD_PROVIDER=digitalocean
DO_TOKEN=your_do_token
```

## Usage

Run tests:

```
go run cmd/main.go
go run cmd/main.go --cluster-name my-test
```

Destroy cluster:

```
go run cmd/main.go --cluster-name my-test --destroy
```

Use a custom manifest:

```
go run cmd/main.go --manifest path/to/manifest.yaml
```

## Project structure

```
cmd/main.go              - main test orchestration
pkg/config/              - env config loading
pkg/kubectl/             - kubectl wrapper (apply, wait, logs, exec)
pkg/rancher/             - rancher API client
pkg/terraform/           - terraform wrapper + run state
terraform/digitalocean/  - terraform config for DigitalOcean
manifests/               - test manifests
```

## State

A `run_state.json` file is written after cluster creation and upgrade steps. On re-run, completed steps are skipped. Delete this file manually if you want a full re-run from scratch.

## Supported providers

- DigitalOcean
- AWS (planned)
- Azure (planned)
