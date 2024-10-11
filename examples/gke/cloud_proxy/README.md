# Example of GKE cluster connected to CAST AI using castai-cloud-proxy

This Terraform module provisions a GKE cluster managed by CAST AI, using the [castai-cloud-proxy](https://github.com/castai/cloud-proxy) to enable CAST AI to control the GKE cluster without requiring direct permissions to the Google Cloud Project on the CAST AI side. The castai-cloud-proxy runs within the GKE cluster itself, utilizing GCP Workload Identity for the necessary permissions.

This example configuration should be analysed in the following order:
1. Create VPC - [`vpc.tf`](./vpc.tf)
2. Create GKE cluster - [`gke.tf`](./gke.tf)
3. Create CAST AI related resources to connect GKE cluster to CAST AI - [`castai.tf`](./castai.tf)

## Usage

> [!NOTE]  
> You can also ask CAST AI support to enable the GKE API Proxy feature for the whole organization before. In this case, you can ignore steps 3-5 and set `cluster_read_only = false` for the first `terraform apply`.

1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your project name, cluster name, cluster region and CAST AI API token.
3. Initialize Terraform. Under example root folder run:
    ```
    terraform init
    ```
4. Run Terraform apply:
    ```
    terraform apply -var-file=tf.vars
    ```

5. Ask CAST AI support to enable the GKE API Proxy feature for this cluster or organization.
6. Update `tf.vars` file and set `cluster_read_only = false`, after the GKE API Proxy feature was enabled for the cluster.
7. Rerun Terraform apply:
    ```
    terraform apply -var-file=tf.vars
    ```

    The cluster should be now connected in the CAST AI console.

8. To destroy resources created by this example:
    ```
    terraform destroy -var-file=tf.vars
    ```

Please refer to this guide if you run into any issues: https://docs.cast.ai/docs/terraform-troubleshooting.

