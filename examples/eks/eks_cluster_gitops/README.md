## EKS and CAST AI example for GitOps onboarding flow

GitOps flow 
===================
Terraform Managed ==> IAM roles, CAST AI Node Configuration, CAST Node Templates and CAST Autoscaler policies

Helm Managed ==> All Castware components such as `castai-agent`, `castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner`, `castai-egressd` are to be installed using other means (e.g ArgoCD, manual Helm releases, etc.)

                                                    +-------------------------+
                                                    |         Start           |
                                                    +-------------------------+
                                                                |
                                                                | AWS CLI
                                                    +-------------------------+
                                                    | 1.Check EKS Auth Mode is API/API_CONFIGMAP
                                                    | 
                                                    +-------------------------+
                                                                |
                                                                | 
                                        -----------------------------------------------------
                                        | YES                                               | NO
                                        |                                                   |
                            +-------------------------+                      +-----------------------------------------+
                            No action needed from User                     2. User to add cast role in aws-auth configmap
                            
                            +-------------------------+                      +-----------------------------------------+
                                        |                                                   |
                                        |                                                   |
                                        -----------------------------------------------------
                                                                | 
                                                                | TERRAFORM             
                                                    +-------------------------+
                                                    |  3. Update TF.VARS      | 
                                                    +-------------------------+
                                                                | 
                                                                | TERRAFORM
                                                    +-------------------------+
                                                    | 4. Terraform Init & Apply| 
                                                    +-------------------------+
                                                                | 
                                                                |GITOPS
                                                    +-------------------------+
                                                    | 5. Deploy Helm chart of castai-agent castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner`
                                                    +-------------------------+         
                                                                | 
                                                                | 
                                                    +-------------------------+
                                                    |         END             |
                                                    +-------------------------+


Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access

> **Note**
>
> used by CAST AI has to be present. Example of entry can be found [here](https://github.com/castai/terraform-provider-castai/blob/157babd57b0977f499eb162e9bee27bee51d292a/examples/eks/eks_cluster_autoscaler_polices/eks.tf#L28-L38).


###### For Step 1: 
# Get EKS cluster authentication mode
CLUSTER_NAME=""
REGION=""
current_auth_mode=$(aws eks describe-cluster --name $CLUSTER_NAME --region $REGION | grep authenticationMode | awk '{print $2}')
echo "Authentication mode is $current_auth_mode"


###### For Step 2: If EKS AUTH mode is API/API_CONFIGMAP, This step can be SKIPPED.
# User to add cast role in aws-auth configmap



###### For Step 5: Deploy Helm chart of castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner`

CASTAI_API_TOKEN=""
CASTAI_CLUSTER_ID=""

# Castai-agent
helm upgrade -i castai-agent castai-helm/castai-agent -n castai-agent \
  --set apiKey=$CASTAI_API_TOKEN \
  --set provider=eks \
  --create-namespace

# castai-cluster-controller
helm upgrade -i cluster-controller castai-helm/castai-cluster-controller -n castai-agent \
--set castai.apiKey=$CASTAI_API_TOKEN \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set autoscaling.enabled=true

# castai-spot-handler
helm upgrade -i castai-spot-handler castai-helm/castai-spot-handler -n castai-agent \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set castai.provider=aws

# castai-evictor
helm upgrade -i castai-evictor castai-helm/castai-evictor -n castai-agent --set replicaCount=0

# castai-pod-pinner
helm upgrade -i castai-pod-pinner castai-helm/castai-pod-pinner -n castai-agent \
--set castai.apiKey=$CASTAI_API_TOKEN \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set replicaCount=0

# castai-kvisor
helm upgrade -i castai-kvisor castai-helm/castai-kvisor -n castai-agent \
--set castai.apiKey=$CASTAI_API_TOKEN \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set controller.extraArgs.kube-linter-enabled=true \
--set controller.extraArgs.image-scan-enabled=true \
--set controller.extraArgs.kube-bench-enabled=true \
--set controller.extraArgs.kube-bench-cloud-provider=eks

1. Configure `tf.vars.example` file with required values. If EKS cluster is already managed by Terraform you could instead directly reference those resources.
2. Run `terraform init`
3. Run `terraform apply` and make a note of `cluster_id`  output values. At this stage you would see that your cluster is in `Connecting` state in CAST AI console
4. Install CAST AI components using Helm. Use `cluster_id` and `api_key` values to configure Helm releases:
- Set `castai.apiKey` property to `api_key` for following CAST AI components: `castai-cluster-controller`, `castai-kvisor`.
- Set `additionalEnv.STATIC_CLUSTER_ID` property to `cluster_id` and `apiKey` property to `api_key` for `castai-agent`.
- Set `castai.clusterID` property to for `castai-cluster-controller`, `castai-spot-handler`, `castai-kvisor`


5. Update [aws-auth](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) configmap with instance profile used by CAST AI. This instance profile is used by CAST AI managed nodes to communicate with EKS control plane.  Example of entry can be found [here](https://github.com/castai/terraform-provider-castai/blob/157babd57b0977f499eb162e9bee27bee51d292a/examples/eks/eks_cluster_assumerole/eks.tf#L28-L38).
6. After all CAST AI components are installed in the cluster its status in CAST AI console would change from `Connecting` to `Connected` which means that cluster onboarding process completed successfully.


## Importing already onboarded cluster to Terraform

This example can also be used to import EKS cluster to Terraform which is already onboarded to CAST AI console trough [script](https://docs.cast.ai/docs/cluster-onboarding#how-it-works).   
For importing existing cluster follow steps 1-3 above and change `castai_node_configuration.default` Node Configuration name.
This would allow to manage already onboarded clusters' CAST AI Node Configurations and Node Templates through IaC.