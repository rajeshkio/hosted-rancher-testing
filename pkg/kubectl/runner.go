package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Runner struct {
	kubeconfigPath string
	kubectlBin     string
}

// NewRunner initializes the runner and verifies kubectl exists to avoid 127 errors.
func NewRunner(kubeconfigContent string) (*Runner, error) {
	// 1. Verify kubectl is installed
	path, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, fmt.Errorf("kubectl binary not found in PATH: %w", err)
	}

	// 2. Setup Kubeconfig
	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tempFile.Write([]byte(kubeconfigContent)); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, fmt.Errorf("write kubeconfig: %w", err)
	}
	_ = tempFile.Close()

	return &Runner{
		kubeconfigPath: tempFile.Name(),
		kubectlBin:     path,
	}, nil
}

// Apply handles manifest application with a standard timeout.
func (r *Runner) Apply(ctx context.Context, manifestPath string) error {
	cmd := exec.CommandContext(ctx, r.kubectlBin, "apply", "-f", manifestPath, "--kubeconfig", r.kubeconfigPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apply failed: %s: %w", stderr.String(), err)
	}
	return nil
}

// GetAllUnhealthyPods identifies pods NOT in Running/Succeeded phase OR Running but not Ready (CrashLoop).
func (r *Runner) GetAllUnhealthyPods(ctx context.Context) ([]string, error) {
	// JSONPath: namespace/name status ready_status
	// We check if the 'ready' field exists for each container status
	format := "-o=jsonpath={range .items[*]}{.metadata.namespace}/{.metadata.name} {.status.phase} {.status.containerStatuses[*].ready}{\"\\n\"}{end}"

	cmd := exec.CommandContext(ctx, r.kubectlBin, "get", "pods", "-A", format, "--kubeconfig", r.kubeconfigPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("get pods failed: %s: %w", stderr.String(), err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var unhealthy []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		podRef := fields[0]
		phase := fields[1]

		// Condition 1: Phase is not a terminal success or active healthy state
		if phase != "Running" && phase != "Succeeded" {
			unhealthy = append(unhealthy, podRef)
			continue
		}

		// Condition 2: Phase is Running, but "false" exists in container ready statuses (CrashLoop/ImagePull)
		if phase == "Running" && strings.Contains(line, "false") {
			unhealthy = append(unhealthy, podRef)
		}
	}
	return unhealthy, nil
}

// GetPods uses jsonpath to return names safely without "pod/" prefixes.
func (r *Runner) GetPods(ctx context.Context, namespace, labelSelector string) ([]string, error) {
	cmd := exec.CommandContext(ctx, r.kubectlBin, "get", "pods", "-n", namespace, "-l", labelSelector, "-o=jsonpath={.items[*].metadata.name}", "--kubeconfig", r.kubeconfigPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("kubectl get failed: %s: %w", stderr.String(), err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}
	return strings.Fields(output), nil
}

// WaitForPod uses the native kubectl wait logic.
func (r *Runner) WaitForPod(ctx context.Context, namespace, podName string) error {
	// Note: timeout is handled by the Go Context

	cmd := exec.CommandContext(ctx, r.kubectlBin, "wait", "--for=condition=ready", "pod/"+podName, "-n", namespace, "--kubeconfig", r.kubeconfigPath)
	fmt.Printf("cmd: %s", cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wait failed: %s: %w", stderr.String(), err)
	}
	return nil
}

func (r *Runner) WaitForAllPodsReady(ctx context.Context) error {
	for {
		unhealthy, err := r.GetAllUnhealthyPods(ctx)
		if err != nil {
			return err
		}
		if len(unhealthy) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for pods: %v", unhealthy)
		case <-time.After(10 * time.Second):
		}
	}
}
func (r *Runner) Logs(ctx context.Context, namespace, podName string, tailLines int) (string, error) {
	args := []string{"logs", podName, "-n", namespace, fmt.Sprintf("--tail=%d", tailLines), "--kubeconfig", r.kubeconfigPath}
	cmd := exec.CommandContext(ctx, r.kubectlBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("logs failed: %s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

// Exec executes a command inside a pod and returns combined output.
func (r *Runner) Exec(ctx context.Context, namespace, podName string, command []string) (string, error) {
	args := append([]string{"exec", podName, "-n", namespace, "--kubeconfig", r.kubeconfigPath, "--"}, command...)
	cmd := exec.CommandContext(ctx, r.kubectlBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("exec failed: %s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

func (r *Runner) GetNodeVersions(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, r.kubectlBin, "get", "nodes",
		"-o=jsonpath={range .items[*]}{.metadata.name}={.status.nodeInfo.kubeletVersion}{\"\\n\"}{end}",
		"--kubeconfig", r.kubeconfigPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("get node versions failed: %s: %w", stderr.String(), err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var versions []string
	for _, line := range lines {
		if line != "" {
			versions = append(versions, line)
		}
	}
	return versions, nil
}

// Cleanup removes the temporary kubeconfig.
func (r *Runner) Cleanup() error {
	if r.kubeconfigPath != "" {
		return os.Remove(r.kubeconfigPath)
	}
	return nil
}
