package rancher

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/norman/clientbase"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type Client struct {
	URL    string
	client *managementClient.Client
}

func NewClient(url, token string) (*Client, error) {
	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("https://%s", url)
	}

	if !strings.HasSuffix(url, "/v3") {
		url = strings.TrimSuffix(url, "/") + "/v3"
	}

	opts := &clientbase.ClientOpts{
		URL:      url,
		TokenKey: token,
		Insecure: true,
	}

	client, err := managementClient.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create rancher client: %w", err)
	}

	return &Client{
		URL:    url,
		client: client,
	}, nil
}

func (c *Client) VerifyLogin() error {
	_, err := c.client.Cluster.List(nil)
	if err != nil {
		return fmt.Errorf("verify login failed: %w", err)
	}
	return nil
}

func (c *Client) GetKubeconfig(clusterID string) (string, error) {
	cluster, err := c.client.Cluster.ByID(clusterID)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster %s: %w", clusterID, err)
	}

	resp, err := c.client.Cluster.ActionGenerateKubeconfig(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to generate kubeconfig for cluster %s: %w", clusterID, err)
	}
	//	fmt.Println(resp.Config)
	return resp.Config, nil
}

func (c *Client) GetCluster(clusterID string) (*managementClient.Cluster, error) {
	cluster, err := c.client.Cluster.ByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s: %w", clusterID, err)
	}

	fmt.Printf("  Cluster ID: %s\n", cluster.ID)
	return cluster, nil
}

func (c *Client) WaitForClusterReady(clusterID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 30 * time.Second

	for time.Now().Before(deadline) {
		cluster, err := c.client.Cluster.ByID(clusterID)
		if err != nil {
			fmt.Printf(" Warning: error polling cluster: %v (retrying...)\n", err)
			time.Sleep(pollInterval)
			continue
		}

		fmt.Printf(" Cluster state: %s | transitioning: %s\n", cluster.State, cluster.TransitioningMessage)

		if cluster.State == "active" && cluster.Transitioning != "yes" {
			return nil
		}

		time.Sleep(pollInterval)
	}
	return fmt.Errorf("cluster %s did not become active within %v", clusterID, timeout)
}
