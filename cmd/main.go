package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"rajeskio/rancher-tests/pkg/config"
	"rajeskio/rancher-tests/pkg/kubectl"
	"rajeskio/rancher-tests/pkg/rancher"
	"rajeskio/rancher-tests/pkg/terraform"
	"strings"
	"syscall"
)

func main() {

	clusterNameFlag := flag.String("cluster-name", "", "Cluster name (default: rancher-test)")
	manifestPath := flag.String("manifest", "manifests/nginx.yaml", "Path to test manifest")
	//destroyFlag := flag.Bool("destroy", false, "Destroy cluster after tests")
	flag.Parse()

	var clusterName string
	if *clusterNameFlag != "" {
		clusterName = *clusterNameFlag
	} else {
		clusterName = "rancher-test"
		fmt.Println("  Using default cluster name: rancher-test")
		fmt.Println("   Use --cluster-name flag to specify a different name")
		fmt.Println("   Example: go run cmd/main.go --cluster-name my-test")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	var tfRunner *terraform.Runner
	var clusterCreated bool

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n Interrupt received (Ctrl+C)")
		fmt.Println("Terraform may still be running...")

		if clusterCreated && tfRunner != nil {
			fmt.Println("\n WARNING: Cluster resources were created")
			fmt.Println("To clean up run:")
			fmt.Printf("  cd %s && terraform destroy --auto-approve\n", tfRunner.WorkDir)
		}

		fmt.Println("\nExisting...")
		cancel()
		os.Exit(1)
	}()

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
	tfRunner = terraform.NewRunner("./terraform", cfg.Provider)

	if err := tfRunner.Init(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Step 5: Preparing cluster configuration ===")
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

	fmt.Println("\n=== Step 8: Getting the kubeconfig ===")
	kubeconfig, err := client.GetKubeconfig(outputs.ClusterID)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Cluster created but couldn't get kubeconfig")
		os.Exit(1)
	}

	fmt.Println(" kubeconfig obtained")

	fmt.Println("\n=== Step 9: Setting up kubectl ===")
	k8s, err := kubectl.NewRunner(kubeconfig)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer k8s.Cleanup()

	fmt.Println("\n===Step 10: Deplying test application ===")
	if err := k8s.Apply(*manifestPath); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("Application deployed")

	fmt.Println("\n===Step 11: Waiting for pod to be ready ===")
	pods, err := k8s.GetPods("test-app", "app=nginx")
	if err != nil || len(pods) == 0 {
		fmt.Println("Error: No pods found")
		os.Exit(1)
	}

	fmt.Printf("Found pod: %s\n", pods[0])
	if err := k8s.WaitForPod("test-app", pods[0], 120); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Step 12: Testing pod logs ===")
	logs, err := k8s.Logs("test-app", pods[0], 10)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("logs retrieved (%d bytes)\n", len(logs))
	fmt.Println("First few lines:")
	fmt.Println(logs)

	fmt.Println("\n=== Step 13: Testing pod exec ===")
	output, err := k8s.Exec("test-app", pods[0], []string{"nginx", "-v"})
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Printf("Exec successful: %s\n", output)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("ALL TESTS PASSED!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\nCluster: %s\n", clusterName)
	fmt.Printf("Cluster ID: %s\n", outputs.ClusterID)
	fmt.Printf("Provider: %s\n", outputs.Provider)
	fmt.Println("\nTests completed:")
	fmt.Println("  Cluster provisioning")
	fmt.Println("  Kubeconfig access")
	fmt.Println("  Application deployment")
	fmt.Println("  Pod logs")
	fmt.Println("  Pod exec")
	fmt.Println("\nTo destroy:")
	fmt.Printf("  go run cmd/main.go --cluster-name %s --destroy\n", clusterName)

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
