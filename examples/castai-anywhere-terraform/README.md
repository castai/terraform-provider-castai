# CAST AI Anywhere Terraform Deployment

This repository contains Terraform code to deploy CAST AI components on an "anywhere" (on-prem/Minikube) Kubernetes cluster. The configuration deploys two components:

- **CAST AI Agent** 
- **CAST AI Cluster Controller**
- **CAST AI Evictor
- **CAST AI Pod Mutator
- **CAST AI Workload Autoscaler

## Prerequisites

Before deploying, ensure you have the following installed and configured:

- **Terraform** (v1.x recommended)  
  [Download Terraform](https://www.terraform.io/downloads)
- **Minikube**  
  [Start Minikube](https://minikube.sigs.k8s.io/docs/start/)
- **Docker Desktop** (make sure Docker is running)  
  [Download Docker Desktop](https://www.docker.com/products/docker-desktop)
- **kubectl**  
  [Install kubectl](https://kubernetes.io/docs/tasks/tools/)
- **Helm**  
  [Install Helm](https://helm.sh/docs/intro/install/)

You will also need:
- A valid **CAST AI API Key** (https://docs.cast.ai/docs/authentication)
- A unique cluster identifier (for example, `minikube-anywhere-cluster`)

## Repository Structure

- **main.tf**  
  Contains the Terraform configuration to:
  - Start Minikube and wait until it is ready.
  - Configure the Kubernetes and Helm providers (using the `minikube` context).
  - Create the required namespace.
  - Deploy the CAST AI Components.
- **variables.tf**  
  Defines variables for CAST AI API key, cluster identifier, and configurations.
- **outputs.tf**  
  Displays outputs such as the status of the deployed components.

## Setup and Deployment

### 1. Clone the Repository

Clone the repository to your local machine:

```sh
git clone https://github.com/juliette-cast/castai-anywhere-terraform.git
cd castai-anywhere-terraform
```

### 2. Initialize Terraform

Run the following command to initialize Terraform and download the required providers:

```sh
terraform init
```

### 3. Validate the Terraform Configuration

Ensure the configuration is correct:

```sh
terraform validate
```

### 4. Plan the Deployment

Preview what Terraform will create:

```sh
terraform plan
```

### 5. 1st Apply the Configuration

This will create the cluster and Deploy the CAST AI Agentand connect to the UI, then you will use the clustr ID from the console to add to your vaoraibles to deploy the other components

```sh
terraform apply
```

### 6. 2nd Apply to deploy the other components

```sh
terraform apply
```

### 7. Verify Deployment

Check the status of deployed components:

```sh
kubectl get pods -n castai-agent
```

Expected output:
```sh
NAME                                          READY   STATUS    RESTARTS   AGE
castai-agent-79bf777cc8-8w88l                 2/2     Running   0             22m
castai-agent-79bf777cc8-kvf2q                 2/2     Running   0             22m
castai-agent-cpvpa-964fc94b6-pqfzc            1/1     Running   0             23m
castai-cluster-controller-77dffcd8f5-7jflv    2/2     Running   0             19m
castai-cluster-controller-77dffcd8f5-cpnp6    2/2     Running   0             19m
castai-evictor-64bdd9fb6c-tmxjv               1/1     Running   0             37s
castai-evictor-cpvpa-6c6bdf8f74-r2m4b         1/1     Running   0             37s
castai-pod-mutator-7556c5db85-pqrwx           1/1     Running   0             16m
castai-pod-mutator-7556c5db85-tqzsh           1/1     Running   1             16m
castai-workload-autoscaler-64655596c4-b72l7   1/1     Running   0             15m
castai-workload-autoscaler-64655596c4-x7bj8   1/1     Running   1 (15m ago)   15m
```

### 7. Check CAST AI Console

1. **Log in to [CAST AI Console](https://app.cast.ai)**
2. **Navigate to `Clusters`**
3. **Confirm that the Minikube cluster is "Connected"**

### 8. Optional: Destroy the Deployment

If you need to remove the deployment, run:

```sh
terraform destroy
```
or delete the castai-agent namespace
```sh
kubectl delete ns castai-agent
```
