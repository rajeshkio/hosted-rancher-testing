package main

import (
	"fmt"
	"os"
	"rajeskio/rancher-tests/pkg/config"
	"rajeskio/rancher-tests/pkg/rancher"
	"rajeskio/rancher-tests/pkg/terraform"
	"time"
)

func main() {
	fmt.Println("=== Step 1: Reading configuration ===")
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Step 2: Connecting to Rancher ===")
	client, err := rancher.NewClient(cfg.RancherURL, cfg.Token)
	if err != nil {
		fmt.Println("Error connecting to Rancher:", err)
		os.Exit(1)
	}

	err = client.VerifyLogin()
	if err != nil {
		fmt.Println("Error verifying login:", err)
		os.Exit(1)
	}
	fmt.Println("Connected to Rancher successfully:", client.URL)

	fmt.Println("\n=== Step 3: Checking cloud provider credentials")
	providerVars, err := getProviderVars(cfg.Provider)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("%s credentials configured\n", cfg.Provider)

	fmt.Println("\n=== Step 4: Initializing Terraform ===")
	tfRunner := terraform.NewRunner("./terraform", cfg.Provider)

	if err := tfRunner.Init(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Step 5: Preparing cluster configuration ===")
	clusterName := fmt.Sprintf("test-%d", time.Now().Unix())
	fmt.Printf("Cluster name: %s\n", clusterName)

	if err := tfRunner.WriteTfvars(cfg.RancherURL, cfg.Token, cfg.K3sVersion, clusterName, providerVars); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Step 6: Create cluster
	fmt.Println("\n=== Step 6: Creating downstream cluster ===")
	if err := tfRunner.Apply(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Step 7: Get cluster details
	fmt.Println("\n=== Step 7: Checking cluster details ===")
	outputs, err := tfRunner.GetOutputs()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("Cluster ID: %s\n", outputs.ClusterID)
	fmt.Printf("Cluster Name: %s\n", outputs.ClusterName)
	fmt.Printf("Provider: %s\n", outputs.Provider)

	fmt.Println("\n Cluster created successfully!")
	fmt.Printf("\nTo destroy: cd terraform/%s && terraform destroy\n", cfg.Provider)

}

func getProviderVars(provider string) (map[string]string, error) {
	vars := make(map[string]string)

	switch provider {
	case "digitalocean":
		doToken := os.Getenv("DO_TOKEN")
		if doToken == "" {
			return nil, fmt.Errorf("DO_TOKEN environment variable not set")
		}
		vars["do_token"] = doToken

		if region := os.Getenv("DO_REGION"); region != "" {
			vars["do_region"] = region
		}
		if size := os.Getenv("DO_SIZE"); size != "" {
			vars["do_size"] = size
		}
	case "aws":
		// FUTURE: AWS credentials
		return nil, fmt.Errorf("AWS provider not implemented yet")

	case "azure":
		// FUTURE: Azure credentials
		return nil, fmt.Errorf("Azure provider not implemented yet")

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	return vars, nil
}
