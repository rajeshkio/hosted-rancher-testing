package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rajeshkio/hosted-rancher-testing/pkg/config"
	"github.com/rajeshkio/hosted-rancher-testing/pkg/kubectl"
	"github.com/rajeshkio/hosted-rancher-testing/pkg/rancher"
	"github.com/rajeshkio/hosted-rancher-testing/pkg/terraform"
)

func main() {

	clusterNameFlag := flag.String("cluster-name", "", "Cluster name (default: rancher-test)")
	manifestPath := flag.String("manifest", "manifests/nginx.yaml", "Path to test manifest")
	destroyFlag := flag.Bool("destroy", false, "Destroy cluster after tests")
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
	fmt.Println("\n=== Step 5: Preparing cluster configuration ===")
	state := terraform.LoadState()
	fmt.Printf("  cluster_deployed=%v cluster_upgraded=%v\n", state.ClusterDeployed, state.ClusterUpgraded)

	if *destroyFlag {
		fmt.Println("\n=== Destroy Mode ===")
		if err := tfRunner.WriteTfvars(cfg.RancherURL, cfg.Token, cfg.K3sVersion, clusterName, providerVars); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		if err := tfRunner.Destroy(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		terraform.ClearState()
		fmt.Println("Cluster destroyed")
		return
	}

	fmt.Println("\n=== Step 6: Creating downstream cluster ===")
	if !state.ClusterDeployed {
		if err := tfRunner.WriteTfvars(cfg.RancherURL, cfg.Token, cfg.K3sVersion, clusterName, providerVars); err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		if err := tfRunner.Apply(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		state.ClusterDeployed = true
		state.CurrentVersion = cfg.K3sVersion
		if err := terraform.SaveState(state); err != nil {
			fmt.Println("Warning: could not save state:", err)
		}
	} else {
		fmt.Println("  Skipping, cluster already deployed")
	}

	fmt.Println("\n=== Step 7: Checking cluster details ===")
	outputs, err := tfRunner.GetOutputs()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if state.ClusterID == "" {
		state.ClusterID = outputs.ClusterID
		terraform.SaveState(state)
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

	// --- Step 10: Deploying test application ---
	fmt.Println("\n=== Step 10: Deploying test application ===")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := k8s.Apply(ctx, *manifestPath); err != nil {
		fmt.Println("Error deploying manifest:", err)
		os.Exit(1)
	}
	fmt.Println("Application deployed")

	// --- Global Health Check ---
	fmt.Println("\n=== Checking for Unhealthy Pods (Cluster-wide) ===")
	// Use a fresh context for health check
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer healthCancel()
	fmt.Println("Waiting for all cluster pods to reach Ready state...")
	if err := k8s.WaitForAllPodsReady(healthCtx); err != nil {
		fmt.Printf("\nTEST FAILED: Pods are not ready: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("All pods are healthy/running.")

	fmt.Println("\n=== Step 11: Waiting for test-app pod to be ready ===")
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer waitCancel()

	pods, err := k8s.GetPods(waitCtx, "test-app", "app=nginx")
	if err != nil || len(pods) == 0 {
		fmt.Println("Error: No pods found in namespace test-app")
		os.Exit(1)
	}

	fmt.Printf("Found pod: %s. Waiting for 'Ready' condition...\n", pods[0])
	if err := k8s.WaitForPod(waitCtx, "test-app", pods[0]); err != nil {
		fmt.Println("Error waiting for pod:", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Step 12: Testing pod logs ===")
	logCtx, logCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer logCancel()

	logs, err := k8s.Logs(logCtx, "test-app", pods[0], 10)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Printf(" logs retrieved (%d bytes)\n", len(logs))

	fmt.Println("\n=== Step 13: Testing pod exec ===")
	execCtx, execCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer execCancel()

	output, err := k8s.Exec(execCtx, "test-app", pods[0], []string{"nginx", "-v"})
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Printf("Exec successful: %s\n", output)

	if cfg.K3sUpgradeVersion != "" {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("KUBERNTES UPGRADE TEST")
		fmt.Println(strings.Repeat("=", 50))

		fmt.Println("\n=== Step 14: Current cluster Kubernetes version ===")
		preCluster, err := client.GetCluster(outputs.ClusterID)
		if err != nil {
			fmt.Println("Error getting cluster:", err)
			os.Exit(1)
		}
		if preCluster.K3sConfig != nil {
			fmt.Printf(" Current version: %s\n", preCluster.K3sConfig.Version)
		}
		fmt.Printf(" Upgrade target: %s\n", cfg.K3sUpgradeVersion)

		fmt.Println("\n=== Step 15: Triggering Kubernetes version upgrade ===")
		if !state.ClusterUpgraded {
			if err := tfRunner.WriteTfvars(cfg.RancherURL, cfg.Token, cfg.K3sUpgradeVersion, clusterName, providerVars); err != nil {
				fmt.Println("Error writing updated tfvars:", err)
				os.Exit(1)
			}

			if err := tfRunner.Apply(); err != nil {
				fmt.Println("Error applying terraform upgrade:", err)
				os.Exit(1)
			}
			fmt.Println("Upgrade apply completed")

			fmt.Println("\n=== Step 16: Waiting for cluster upgrade to complete ===")
			fmt.Println("This may take 10-15 minutes ...")
			if err := client.WaitForClusterReady(outputs.ClusterID, 15*time.Minute); err != nil {
				fmt.Println("Error waiting for upgrade:", err)
				os.Exit(1)
			}
			fmt.Println("Cluster upgrade completed")
		} else {
			fmt.Println("  Skipping, cluster already upgraded")
		}

		fmt.Println("\n === Step 17: Re-fetching kubeconfig after upgrade ===")
		k8s.Cleanup()
		newKubeconfig, err := client.GetKubeconfig(outputs.ClusterID)
		if err != nil {
			fmt.Println("Error getting kubeconfig after upgrade:", err)
			os.Exit(1)
		}

		k8s, err = kubectl.NewRunner(newKubeconfig)
		if err != nil {
			fmt.Println("Error setiing up kubectl: ", err)
			os.Exit(1)
		}

		defer k8s.Cleanup()
		fmt.Println("kubeconfig refreshed")

		fmt.Println("\n === Step 18: Verifying node Kubernetes versions ===")
		nodeCtx, nodeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer nodeCancel()

		nodeVersions, err := k8s.GetNodeVersions(nodeCtx)
		if err != nil {
			fmt.Println("Error getting node versions:", err)
			os.Exit(1)
		}
		for _, nv := range nodeVersions {
			fmt.Printf("  %s\n", nv)
		}

		fmt.Println("\n=== Step 19: Post-upgrade health check ===")
		postHealthCtx, postHealthCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer postHealthCancel()

		fmt.Println("Waiting for all cluster pods to reach Ready state...")
		if err := k8s.WaitForAllPodsReady(postHealthCtx); err != nil {
			fmt.Printf("\nUPGRADE TEST FAILED: Pods did not stabilize after upgrade")
			os.Exit(1)
		}
		fmt.Println("All pods are healthy after upgrade")

		// Re-verify the test app pod
		postWaitCtx, postWaitCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer postWaitCancel()

		postPods, err := k8s.GetPods(postWaitCtx, "test-app", "app=nginx")
		if err != nil || len(postPods) == 0 {
			fmt.Println("Error: test-app pods not found after upgrade")
			os.Exit(1)
		}
		fmt.Printf("Test app pod %s still running after upgrade\n", postPods[0])

		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("UPGRADE TEST PASSED!")
		fmt.Printf("  %s -> %s\n", cfg.K3sVersion, cfg.K3sUpgradeVersion)
		fmt.Println(strings.Repeat("=", 50))
	}

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
	if cfg.K3sUpgradeVersion != "" {
		fmt.Printf("  Kubernetes upgrade (%s -> %s)\n", cfg.K3sVersion, cfg.K3sUpgradeVersion)
	}
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
