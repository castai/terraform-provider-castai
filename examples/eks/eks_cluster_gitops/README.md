## EKS and CAST AI example for GitOps onboarding flow

Following example shows how to onboard EKS cluster to CAST AI using GitOps flow.
In GitOps flow CAST AI Node Configuration, Node Templates and Autoscaler policies are managed using Terraform, but all Castware components such as `castai-agent`, `castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor` are to be installed using other means (e.g ArgoCD, manual Helm releases, etc.)

Steps to take to successfully onboard EKS cluster to CAST AI using GitOps flow:

Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

1. Configure `tf.vars.example` file with required values. If EKS cluster is already managed by Terraform you could instead directly reference those resources.
2. Run `terraform init`
3. Run `terraform apply` and make a note of `cluster_id` and `cluster_token` output values. At this stage you would see that your cluster is in `Connecting` state in CAST AI console
4. Install CAST AI components using Helm. Use `cluster_id` and `cluster_token` values to configure Helm releases:
- Set `castai.apiKey` property to `cluster_token` for following CAST AI components: `castai-cluster-controller`, `castai-kvisor`.
- Set `additionalEnv.STATIC_CLUSTER_ID` property to `cluster_id` and `apiKey` property to `cluster_token` for `castai-agent`.
- Set `castai.clusterID` property to for `castai-cluster-controller`, `castai-spot-handler`, `castai-kvisor`
Example Helm install command:

```bash
helm install castai-cluster-controller cluster-controller --namespace=castai-agent --set castai.apiKey=<cluster_token>,provider=eks,castai.clusterID=<cluster_id>,createNamespace=false,apiURL="https://api.cast.ai"
```

5. Update [aws-auth](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) configmap with instance profile used by CAST AI. This instance profile is used by CAST AI managed nodes to communicate with EKS control plane.  Example of entry can be found [here](https://github.com/castai/terraform-provider-castai/blob/157babd57b0977f499eb162e9bee27bee51d292a/examples/eks/eks_cluster_assumerole/eks.tf#L28-L38).
6. After all CAST AI components are installed in the cluster its status in CAST AI console would change from `Connecting` to `Connected` which means that cluster onboarding process completed successfully.


## Importing already onboarded cluster to Terraform

This example can also be used to import EKS cluster to Terraform which is already onboarded to CAST AI console trough [script](https://docs.cast.ai/docs/cluster-onboarding#how-it-works).   
For importing existing cluster follow steps 1-3 above and change `castai_node_configuration.default` Node Configuration name.
This would allow to manage already onboarded clusters' CAST AI Node Configurations and Node Templates through IaC.
