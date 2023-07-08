## CAST AI EKS clusters examples

[Read only EKS cluster](eks_cluster_readonly/) - Onboard EKS cluster to CAST AI in read-only mode.
[Read-only EKS cluster with possible migration to Full access mode](eks_cluster_optional_readonly/) - Onboard EKS cluster to CAST AI in read-only mode with possibility to onboard to Full Access mode by controlling the variable `readonly=true`.
[EKS cluster](eks_cluster_assumerole/) - Onboard EKS cluster to CAST AI in Full Access mode. [CAST AI IAM module](https://github.com/castai/terraform-castai-eks-role-iam) is used to create required IAM resources.
[EKS cluster with custom IAM polices](eks_cluster_custom_iam/) -  Onboard EKS cluster to CAST AI in Full Access mode. Custom IAM policies are used to configure required IAM resources.
[EKS cluster with autoscaler policies](eks_cluster_autoscaler_policies/) -  Onboard EKS cluster to CAST AI in Full Access mode with configured autoscaler CAST AI policies.
[EKS cluster onboarded through GitOps](eks_cluster_gitops/) - Onboard EKS cluster to CAST AI in Full Access mode when CAST AI K8s components are installed using GitOps (e.g. ArgoCD, manual Helm releases). This example can also be used to import CAST AI cluster to Terraform if it was onboarded using [onboarding script](https://docs.cast.ai/docs/cluster-onboarding).
[EKS cluster with demo application](eks_cluster_webshop) - Onboard EKS cluster to CAST AI in Full Access mode. This example contains Demo application installed into cluster to showcase CAST AI cost saving capabilities.
 