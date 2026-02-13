package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	RancherVersion string
	K3sVersion     string
	RancherURL     string
	Token          string
	Provider       string
}

func ReadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	cfg := &Config{}
	cfg.RancherVersion = os.Getenv("RANCHER_VERSION")
	cfg.K3sVersion = os.Getenv("K3S_VERSION")
	cfg.RancherURL = os.Getenv("RANCHER_URL")
	cfg.Token = os.Getenv("RANCHER_TOKEN")
	cfg.Provider = os.Getenv("CLOUD_PROVIDER")
	if cfg.Provider == "" {
		cfg.Provider = "digitalocean"
	}

	var missing []string

	if cfg.RancherVersion == "" {
		missing = append(missing, "RANCHER_VERSION")
	}
	if cfg.K3sVersion == "" {
		missing = append(missing, "K3S_VERSION")
	}
	if cfg.RancherURL == "" {
		missing = append(missing, "RANCHER_URL")
	}
	if cfg.Token == "" {
		missing = append(missing, "RANCHER_TOKEN")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env variables %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}
