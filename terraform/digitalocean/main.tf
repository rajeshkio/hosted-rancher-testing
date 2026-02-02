terraform {
  required_providers {
    rancher2 = {
      source  = "rancher/rancher2"
      version = "13.1.4"
    }
  }
}

provider "rancher2" {
  api_url   = var.rancher_url
  token_key = var.rancher_token
  insecure  = true
}

resource "rancher2_cloud_credential" "do" {
  name = "${var.cluster_name}-cred"

  digitalocean_credential_config {
    access_token = var.do_token
  }
}

resource "rancher2_machine_config_v2" "do_nodes" {
  generate_name = "${var.cluster_name}-do-pool"

  digitalocean_config {
    access_token = var.do_token
    image        = var.do_image
    region       = var.do_region
    size         = var.do_size
  }
}

resource "rancher2_cluster_v2" "downstream"  {
  name = var.cluster_name
  kubernetes_version = var.k3s_version

  rke_config {
    machine_pools {
      name                         = "pool1"
      cloud_credential_secret_name = rancher2_cloud_credential.do.id
      control_plane_role           = true
      etcd_role                    = true
      worker_role                  = true
      quantity                     = var.node_count

      machine_config {
        kind = rancher2_machine_config_v2.do_nodes.kind
        name = rancher2_machine_config_v2.do_nodes.name
      }
    }
  }
}

resource "rancher2_cluster_sync" "downstream" {
  cluster_id = rancher2_cluster_v2.downstream.cluster_v1_id
}