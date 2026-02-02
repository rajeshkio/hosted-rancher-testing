# Rancher Version Testing Tool

Automated tool to test Rancher versions by creating downstream Kubernetes clusters and running validation tests.

## What This Does

1. **Creates a downstream K3s cluster** on DigitalOcean (or other cloud providers)
2. **Deploys a test application** (nginx by default)
3. **Validates cluster functionality**:
   - Kubeconfig access
   - Pod deployment
   - Pod logs retrieval
   - Pod exec commands

This allows you to test different Rancher and K3s versions quickly and consistently.

## Prerequisites

### Required Tools

- **Go** Built and tested on 1.25
- **Terraform** Build and tested on v1.13.4
- **kubectl** (for manual verification)

### Required Accounts

- **Rancher cluster** (already running)
- **DigitalOcean account** with API token

### Required Access

- Rancher API token with cluster creation permissions
- DigitalOcean API token

## Installation

### 1. Clone and Setup

```bash
git clone <your-repo>
cd rancher-tests

# Download Go dependencies
go mod download
```

### 2. Directory Structure

```
rancher-tests/
├── cmd/
│   └── main.go                    # Main entry point
├── pkg/
│   ├── config/
│   │   └── reader.go             # Configuration reader
│   ├── rancher/
│   │   └── client.go             # Rancher API client
│   ├── terraform/
│   │   └── runner.go             # Terraform wrapper
│   └── kubectl/
│       └── runner.go             # kubectl wrapper
├── terraform/
│   └── digitalocean/             # DO provider config
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
├── manifests/
│   └── nginx.yaml                # Test application manifest
├── go.mod
└── README.md
```

## Configuration

### Environment Variables

Set these before running:

```bash
# Required: Rancher configuration
export RANCHER_VERSION="2.13.2"  # Rancher version to deploy in the downstream cluster
export RANCHER_URL="https://your-rancher.example.com"
export RANCHER_TOKEN="token-xxxxx:yyyyy"

# Required: Kubernetes version to test
export K3S_VERSION="v1.31.14+k3s1"

# Required: Cloud provider credentials
export DO_TOKEN="dop_v1_xxxxxxxxxxxxx"

# Optional: Cloud provider (defaults to digitalocean)
export CLOUD_PROVIDER="digitalocean"

# Optional: DigitalOcean settings
export DO_REGION="nyc3"              # Default: nyc3
export DO_SIZE="s-2vcpu-4gb"         # Default: s-2vcpu-4gb
```

### Getting Credentials

**Rancher Token:**

1. Login to Rancher UI
2. Click your user icon → API & Keys
3. Create API Key
4. Copy the token (format: `token-xxxxx:yyyyy`)

**DigitalOcean Token:**

1. Login to DigitalOcean
2. Go to API → Tokens/Keys
3. Generate New Token
4. Copy the token (format: `dop_v1_xxxxx`)

## Usage

### Basic Usage

**Create and test a cluster:**

```bash
export RANCHER_VERSION="2.13.2"
export K3S_VERSION="v1.31.14+k3s1"
export RANCHER_URL="https://your-rancher.example.com"
export RANCHER_TOKEN="token-xxxxx:yyyyy"
export DO_TOKEN="dop_v1_xxxxx"

go run cmd/main.go --cluster-name my-test
```

**What happens:**

1. Connects to Rancher
2. Creates downstream K3s cluster on DigitalOcean (10-15 minutes)
3. Gets kubeconfig
4. Deploys nginx test application
5. Validates logs and exec functionality
6. Shows summary

### Command-Line Flags

```bash
# Specify cluster name (default: rancher-test)
go run cmd/main.go --cluster-name my-cluster

# Use custom test manifest
go run cmd/main.go --manifest manifests/custom-app.yaml

# Destroy cluster
go run cmd/main.go --cluster-name my-cluster --destroy
```

**Cleanup:**

```bash
go run cmd/main.go --cluster-name my-test --destroy
```

## Custom Test Manifests

Create your own test application:

**manifests/custom-app.yaml:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-test
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: my-test
  labels:
    app: my-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: app
          image: your-image:tag
          ports:
            - containerPort: 8080
```

**Use it:**

```bash
go run cmd/main.go --manifest manifests/custom-app.yaml
```

**Note:** Make sure your manifest includes:

- A namespace resource
- Labels (app: your-app-name) e.g app: nginx
- The tool will look for pods with that label

## Output Examples

### Successful Run

```
=== Step 1: Reading configuration ===
Rancher version: 2.13.2
K3s version: v1.31.14+k3s1
Cloud provider: digitalocean
Cluster name: my-test

=== Step 2: Connecting to Rancher ===
✓ Connected to Rancher

=== Step 3: Checking cloud provider credentials ===
✓ digitalocean credentials configured

=== Step 4: Initializing Terraform ===
Running terraform init for digitalocean...
✓ Terraform initialized

=== Step 5: Preparing cluster configuration ===
✓ Terraform variables written

=== Step 6: Creating downstream cluster ===
[Terraform output showing cluster creation progress...]
✓ Terraform apply completed successfully

=== Step 7: Getting cluster details ===
✓ Cluster ID: c-m-abc123
✓ Cluster Name: my-test

=== Step 8: Getting kubeconfig ===
✓ Kubeconfig obtained

=== Step 9: Setting up kubectl ===
✓ Kubeconfig saved to /tmp/kubeconfig-123.yaml

=== Step 10: Deploying test application ===
namespace/test-app created
deployment.apps/nginx created
✓ Application deployed

=== Step 11: Waiting for pod to be ready ===
Found pod: nginx-7c6b8d9f8-abc12
✓ Pod nginx-7c6b8d9f8-abc12 is ready

=== Step 12: Testing pod logs ===
✓ Logs retrieved (450 bytes)

=== Step 13: Testing pod exec ===
✓ Exec successful: nginx version: nginx/1.25.0

==================================================
✅ ALL TESTS PASSED!
==================================================

Cluster: my-test
Cluster ID: c-m-abc123
Provider: digitalocean

Tests completed:
  ✓ Cluster provisioning
  ✓ Kubeconfig access
  ✓ Application deployment
  ✓ Pod logs
  ✓ Pod exec

To destroy:
  go run cmd/main.go --cluster-name my-test --destroy
```

## Troubleshooting

### Issue: "DO_TOKEN not set"

**Solution:**

```bash
export DO_TOKEN="dop_v1_your_token_here"
```

### Issue: "Terraform init failed"

**Solution:**

```bash
# Clean and reinitialize
cd terraform/digitalocean
rm -rf .terraform .terraform.lock.hcl
terraform init
cd ../..
```

### Issue: "Cluster stuck in Provisioning"

**Check:**

1. Rancher UI → Cluster Management → Your cluster
2. Check events and logs
3. Verify DigitalOcean droplets are created (DO console)

**Debug:**

```bash
cd terraform/digitalocean
terraform show  # See current state
```

### Issue: Ctrl+C doesn't clean up

The tool warns you about cleanup. To manually clean up:

```bash
# Option 1: Use the tool
go run cmd/main.go --cluster-name your-cluster --destroy

# Option 2: Manual terraform
cd terraform/digitalocean
terraform destroy -auto-approve
```

### Issue: "No pods found"

**Check manifest:**

- Does it have a namespace?
- Does it have labels (app: your-app)?
- Is the image accessible?

**Verify manually:**

```bash
# Get kubeconfig from Rancher UI and save to file
kubectl --kubeconfig=/path/to/kubeconfig get pods -A
```

## Development

### Adding a New Cloud Provider

1. **Create terraform config:**

```bash
mkdir terraform/aws
# Add main.tf, variables.tf, outputs.tf
```

2. **Update getProviderVars in main.go:**

```go
case "aws":
    vars["aws_access_key"] = os.Getenv("AWS_ACCESS_KEY_ID")
    vars["aws_secret_key"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
    vars["aws_region"] = os.Getenv("AWS_REGION")
```

3. **Test:**

```bash
export CLOUD_PROVIDER="aws"
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
go run cmd/main.go --cluster-name aws-test
```
