## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_rancher2"></a> [rancher2](#requirement\_rancher2) | 13.1.4 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_rancher2"></a> [rancher2](#provider\_rancher2) | 13.1.4 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [rancher2_cloud_credential.do](https://registry.terraform.io/providers/rancher/rancher2/13.1.4/docs/resources/cloud_credential) | resource |
| [rancher2_cluster_sync.downstream](https://registry.terraform.io/providers/rancher/rancher2/13.1.4/docs/resources/cluster_sync) | resource |
| [rancher2_cluster_v2.downstream](https://registry.terraform.io/providers/rancher/rancher2/13.1.4/docs/resources/cluster_v2) | resource |
| [rancher2_machine_config_v2.do_nodes](https://registry.terraform.io/providers/rancher/rancher2/13.1.4/docs/resources/machine_config_v2) | resource |
| [rancher2_setting.agent_tls_mode](https://registry.terraform.io/providers/rancher/rancher2/13.1.4/docs/resources/setting) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Name for the test cluster | `string` | n/a | yes |
| <a name="input_do_image"></a> [do\_image](#input\_do\_image) | DigitalOcean image | `string` | `"ubuntu-24-04-x64"` | no |
| <a name="input_do_region"></a> [do\_region](#input\_do\_region) | DigitalOcean region | `string` | `"nyc3"` | no |
| <a name="input_do_size"></a> [do\_size](#input\_do\_size) | DigitalOcean droplet size | `string` | `"s-4vcpu-8gb"` | no |
| <a name="input_do_token"></a> [do\_token](#input\_do\_token) | DigitalOcean API token | `string` | n/a | yes |
| <a name="input_k3s_version"></a> [k3s\_version](#input\_k3s\_version) | K3s version to install | `string` | n/a | yes |
| <a name="input_node_count"></a> [node\_count](#input\_node\_count) | Number of nodes | `number` | `1` | no |
| <a name="input_rancher_token"></a> [rancher\_token](#input\_rancher\_token) | Rancher API token | `string` | n/a | yes |
| <a name="input_rancher_url"></a> [rancher\_url](#input\_rancher\_url) | Rancher server URL | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_cluster_id"></a> [cluster\_id](#output\_cluster\_id) | Rancher cluster ID |
| <a name="output_cluster_name"></a> [cluster\_name](#output\_cluster\_name) | Cluster name |
| <a name="output_provider"></a> [provider](#output\_provider) | Cloud provider used |
