## GKE and CAST AI example for GitOps onboarding flow

## GitOps flow 

Terraform Managed ==>  IAM roles, CAST AI Node Configuration, CAST Node Templates and CAST Autoscaler policies

Helm Managed ==>  All Castware components such as `castai-agent`, `castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner`, `castai-egressd` are to be installed using other means (e.g ArgoCD, manual Helm releases, etc.)


                                                +-------------------------+
                                                |         Start           |
                                                +-------------------------+
                                                            | 
                                                            | TERRAFORM
                                                +-------------------------+
                                                | 1. Update TF.VARS 
                                                  2. Terraform Init & Apply| 
                                                +-------------------------+
                                                            | 
                                                            | TERRAFORM OUTPUT
                                                +-------------------------+
                                                |  3. Execute terraform output command
                                                | terraform output cluster_id  
                                                  terraform output cluster_token
                                                +-------------------------+
                                                            | 
                                                            |GITOPS
                                                +-------------------------+
                                                | 3. Deploy Helm chart of castai-agent castai-cluster-controller`, `castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner`
                                                +-------------------------+         
                                                            | 
                                                            | 
                                                +-------------------------+
                                                |         END             |
                                                +-------------------------+


Prerequisites:
- CAST AI account
- Obtained CAST AI [API Access key](https://docs.cast.ai/docs/authentication#obtaining-api-access-key) with Full Access


### Step 1 & 2: Update TF vars & TF Init, plan & apply
After successful apply, CAST Console UI will be in `Connecting` state. \
Note generated 'CASTAI_CLUSTER_ID' from outputs

### Step 3: Execute TF output command & save the below output values
terraform output cluster_id  
terraform output cluster_token

Obtained values are needed for next step

### Step 4: Deploy Helm chart of CAST Components
Coponents: `castai-cluster-controller`,`castai-evictor`, `castai-spot-handler`, `castai-kvisor`, `castai-workload-autoscaler`, `castai-pod-pinner` \
After all CAST AI components are installed in the cluster its status in CAST AI console would change from `Connecting` to `Connected` which means that cluster onboarding process completed successfully.

```
CASTAI_API_KEY="<Replace cluster_token>"
CASTAI_CLUSTER_ID="<Replace cluster_id>"
CAST_CONFIG_SOURCE="castai-cluster-controller"

#### Mandatory Component: Castai-agent
helm upgrade -i castai-agent castai-helm/castai-agent -n castai-agent --create-namespace \
  --set apiKey=$CASTAI_API_KEY \
  --set provider=gke \
  --set createNamespace=false

#### Mandatory Component: castai-cluster-controller
helm upgrade -i cluster-controller castai-helm/castai-cluster-controller -n castai-agent \
--set castai.apiKey=$CASTAI_API_KEY \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set autoscaling.enabled=true

#### castai-spot-handler
helm upgrade -i castai-spot-handler castai-helm/castai-spot-handler -n castai-agent \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set castai.provider=gcp

#### castai-evictor
helm upgrade -i castai-evictor castai-helm/castai-evictor -n castai-agent --set replicaCount=1

#### castai-pod-pinner
helm upgrade -i castai-pod-pinner castai-helm/castai-pod-pinner -n castai-agent \
--set castai.apiKey=$CASTAI_API_KEY \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set replicaCount=0

#### castai-workload-autoscaler
helm upgrade -i castai-workload-autoscaler castai-helm/castai-workload-autoscaler -n castai-agent \
--set castai.apiKeySecretRef=$CAST_CONFIG_SOURCE \
--set castai.configMapRef=$CAST_CONFIG_SOURCE \

#### castai-kvisor
helm upgrade -i castai-kvisor castai-helm/castai-kvisor -n castai-agent \
--set castai.apiKey=$CASTAI_API_KEY \
--set castai.clusterID=$CASTAI_CLUSTER_ID \
--set controller.extraArgs.kube-linter-enabled=true \
--set controller.extraArgs.image-scan-enabled=true \
--set controller.extraArgs.kube-bench-enabled=true \
--set controller.extraArgs.kube-bench-cloud-provider=gke
```

## Steps Overview

1. Configure `tf.vars.example` file with required values. If AKS cluster is already managed by Terraform you could instead directly reference those resources.
2. Run `terraform init`
3. Run `terraform apply` and make a note of `cluster_id`  output values. At this stage you would see that your cluster is in `Connecting` state in CAST AI console
4. Install CAST AI components using Helm. Use `cluster_id` and `api_key` values to configure Helm releases:
- Set `castai.apiKey` property to `api_key`
- Set `castai.clusterID` property to `cluster_id`
5. After all CAST AI components are installed in the cluster its status in CAST AI console would change from `Connecting` to `Connected` which means that cluster onboarding process completed successfully.


## Importing already onboarded cluster to Terraform

This example can also be used to import GKE cluster to Terraform which is already onboarded to CAST AI console through [script](https://docs.cast.ai/docs/cluster-onboarding#how-it-works).   
For importing existing cluster follow steps 1-3 above and change `castai_node_configuration.default` Node Configuration name.
This would allow to manage already onboarded clusters' CAST AI Node Configurations and Node Templates through IaC.