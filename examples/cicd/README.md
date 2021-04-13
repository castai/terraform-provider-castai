# CICD cluster

Components:
* Google Could project.
* Google Cloud credentials with required roles.
* CAST AI cluster.
* Cloudflare custom DNS.  
* Gitlab runners with Google Cloud storage buckets for cache.
* ArgoCD.
* Charts Museum to store private helm charts.  
* Pomerium identity aware proxy with Google auth to secure ArgoCD UI.

## Installation

### One time manual steps

#### Variables

Replace local.auto.tfvars file variables with your own.

#### Helm chart values

Check charts helm values and modify for you needs. Replace your-repo and yourdomain.com with your own values.

#### Google OAuth
You need to manually create OAuth client ID and secret in Google Cloud as there is no terraform resources.

* Go to OAuth consent screen and create a screen with external user type.
* Go to Credentials screen and create credentials for OAuth 2.0 Client.
    * Select Web Application type and insert some name.
    * In URIs insert `https://authenticate.argocd.yourdomain.com/oauth2/callback`
    * Save and copy Client ID with Client Secret into terraform variables.

#### ArgoCD target clusters

You need to add target dev-master and prod-master clusters credentials. ArgoCD guide can be found [here](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters).


### Terraform

```
terraform init
```

```
terraform plan
```

```
terraform apply
```
