package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Runner struct {
	WorkDir  string
	Provider string
}

type Output struct {
	ClusterID   string
	ClusterName string
	Provider    string
}

func NewRunner(baseDir, provider string) *Runner {
	workDir := filepath.Join(baseDir, provider)

	return &Runner{
		WorkDir:  workDir,
		Provider: provider,
	}

}

func (r *Runner) Init() error {
	cmd := exec.Command("terraform", "init")
	cmd.Dir = r.WorkDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Println("Running terraform init...")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %s", stderr.String())
	}

	fmt.Println("terraform initialized")
	return nil
}

func (r *Runner) WriteTfvars(rancherURL, rancherToken, k3sVersion, clusterName string, providerVars map[string]string) error {
	tfvarsPath := filepath.Join(r.WorkDir, "terraform.tfvars")

	content := fmt.Sprintf(`rancher_url = "%s"
	rancher_token = "%s"
	k3s_version = "%s"
	cluster_name = "%s"
	`, rancherURL, rancherToken, k3sVersion, clusterName)

	for key, value := range providerVars {
		content += fmt.Sprintf(`%s = "%s"`, key, value)
	}

	if err := os.WriteFile(tfvarsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write tfvars: %w", err)
	}

	fmt.Println("terraform variables written")
	return nil
}

func (r *Runner) Apply() error {
	cmd := exec.Command("terraform", "apply", "--auto-approve")
	cmd.Dir = r.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running terraform apply (this may take 10-15minutes) ....")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %s\n%s", stderr.String(), stdout.String())
	}

	fmt.Println("terraform apply completed")
	return nil
}

func (r *Runner) GetOutputs() (*Output, error) {
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = r.WorkDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("terraform output failed: %w", err)
	}

	var outputs map[string]struct {
		Value string `json:"value"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &outputs); err != nil {
		return nil, fmt.Errorf("parse terraform output: %w", err)
	}

	return &Output{
		ClusterID:   outputs["cluster_id"].Value,
		ClusterName: outputs["cluster_name"].Value,
		Provider:    outputs["provider"].Value,
	}, nil
}

func (r *Runner) Destroy() error {
	cmd := exec.Command("terraform", "destroy", "--auto-approve")
	cmd.Dir = r.WorkDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Println("destroying cluster...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform destroy failed: %s", stderr.String())
	}

	fmt.Println("Cluster destroyed")
	return nil
}
