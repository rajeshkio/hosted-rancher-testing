# Rancher configuration
variable "rancher_url" {
  description = "Rancher server URL"
  type        = string
}

variable "rancher_token" {
  description = "Rancher API token"
  type        = string
  sensitive   = true
}

# Cluster configuration
variable "cluster_name" {
  description = "Name for the test cluster"
  type        = string
}

variable "k3s_version" {
  description = "K3s version to install"
  type        = string
}

variable "node_count" {
  description = "Number of nodes"
  type        = number
  default     = 1
}

# DigitalOcean-specific variables
variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "do_region" {
  description = "DigitalOcean region"
  type        = string
  default     = "nyc3"
}

variable "do_size" {
  description = "DigitalOcean droplet size"
  type        = string
  default     = "s-4vcpu-8gb"
}

variable "do_image" {
  description = "DigitalOcean image"
  type        = string
  default     = "ubuntu-25-04-x64"
}