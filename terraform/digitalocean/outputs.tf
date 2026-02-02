output "cluster_id" {
  description = "Rancher cluster ID"
  value       = rancher2_cluster_v2.downstream.cluster_v1_id
}

output "cluster_name" {
  description = "Cluster name"
  value       = rancher2_cluster_v2.downstream.name
}

output "provider" {
  description = "Cloud provider used"
  value       = "digitalocean"
}

