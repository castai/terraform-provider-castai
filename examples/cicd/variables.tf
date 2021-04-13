variable "castai_api_token" {
  default = ""
}

variable "castai_api_url" {
  default = "https://api.cast.ai/v1"
}

variable "cluster_name" {
  default = "cicd"
}

variable "cluster_region" {
  default = "us-east"
}

variable "gcp_project_name" {
  default = "cicd-master"
}

variable "gcp_org_id" {
  description = "GCP organization id"
  default     = ""
}

variable "gcp_billing_account" {
  description = "GCP billing account"
  default     = ""
}

variable "gcp_region" {
  description = "GCP region, for example: us-east4"
  default     = "us-east4"
}

variable "gcp_credentials" {
  description = "GCP credentials JSON."
}

variable "gcp_auth_client_id" {
  description = "GCP OAuth client id"
}

variable "gcp_auth_client_secret" {
  description = "GCP OAuth client secret"
}

variable "gitlab_runner_registration_token" {
  description = "Registration token used to authenticate with GitLab"
}

variable "argocd_dev_cluster_name" {
  default = "dev-master"
}

variable "argocd_dev_cluster_server" {
  default = ""
  description = "Argo CD dev cluster server address, eg: https://my-server"
}

variable "argocd_dev_cluster_config" {
  default = ""
  description = "Argo CD dev cluster configuration"
}

variable "argocd_prod_cluster_name" {
  default = "prod-master"
}

variable "argocd_prod_cluster_server" {
  description = "Argo CD prod cluster server address, eg: https://my-server"
}

variable "argocd_prod_cluster_config" {
  description = "Argo CD prod cluster configuration"
}

variable "charts_user" {
  description = "BasicAuth username to get write access for charts museum"
}

variable "charts_pass" {
  description = "BasicAuth password to get write access for charts museum"
}

variable "helmChartsSSHPrivateKey" {
  description = "Argo helm charts repository ssh private key"
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token"
}

variable "cloudflare_zone" {
  description = "Cloudflare zone, eg: yourdomain.com"
}

variable "cloudflare_argocd_subdomain" {
  description = "Argocd subdomain name"
  default = "argocd"
}

variable "cloudflare_charts_subdomain" {
  description = "Charts museum subdomain name"
  default = "charts"
}
