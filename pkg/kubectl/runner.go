package kubectl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Runner struct {
	kubeconfigPath string
}

func NewRunner(kubeconfigContent string) (*Runner, error) {

	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tempFile.Write([]byte(kubeconfigContent)); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("write kubeconfig: %w", err)
	}
	tempFile.Close()

	fmt.Printf("kubeconfig saved to %s\n", tempFile.Name())

	return &Runner{
		kubeconfigPath: tempFile.Name(),
	}, nil
}

func (r *Runner) Apply(manifestPath string) error {
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("manifest file not found: %s", manifestPath)
	}

	cmd := exec.Command("kubectl", "apply", "-f", manifestPath, "--kubeconfig", r.kubeconfigPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Applying manifest: %s\n", filepath.Base(manifestPath))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %s\n%s", stderr.String, stdout.String)
	}

	fmt.Println(stdout.String())
	return nil
}

func (r *Runner) GetPods(namespace, labelSelector string) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", labelSelector, "-o", "name", "--kubeconfig", r.kubeconfigPath)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("get pods failed: %w", err)
	}

	output := stdout.String()
	if output == "" {
		return []string{}, nil
	}

	lines := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n"))
	pods := make([]string, 0, len(lines))

	for _, line := range lines {
		podName := string(bytes.TrimPrefix(line, []byte("pod/")))
		if podName != "" {
			pods = append(pods, podName)
		}
	}
	return pods, nil
}

func (r *Runner) WaitForPod(namespace, podName string, timeoutSeconds int) error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=ready", fmt.Sprintf("pod/%s", podName), "-n", namespace, fmt.Sprintf("--timeout=%ds", timeoutSeconds), "--kubeconfig", r.kubeconfigPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Printf("Waiting for pod %s to be ready...\n", podName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wait failed: %s", stderr.String())
	}
	fmt.Printf("Pod %s is ready \n", podName)
	return nil
}

func (r *Runner) Logs(namespace, podName string, tailLines int) (string, error) {
	cmd := exec.Command("kubectl", "logs", podName, "-n", namespace, fmt.Sprintf("--tail=%d", tailLines), "--kubeconfig", r.kubeconfigPath)

	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("get logs failed: %s", stderr.String())
	}

	return stdout.String(), nil
}

func (r *Runner) Exec(namespace, podName string, command []string) (string, error) {

	args := []string{"exec", podName, "-n", namespace, "--kubeconfig", r.kubeconfigPath, "--"}
	args = append(args, command...)
	cmd := exec.Command("kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("exec failed: %s", stderr.String())
	}

	output := stdout.String() + stderr.String()
	return output, nil
}
func (r *Runner) Cleanup() error {
	if r.kubeconfigPath != "" {
		return os.Remove((r.kubeconfigPath))
	}
	return nil
}
