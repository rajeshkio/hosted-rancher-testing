package rancher

import (
	"fmt"
	"strings"

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
