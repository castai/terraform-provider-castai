output "grafana_password" {
  value = random_password.grafana_admin.result

  sensitive = true
}

output "ingress_ips" {
  value = [for ip in aws_eip.this : ip.public_ip]
}

output "loki-arn" {
  value = module.eks_iam_role_s3.service_account_role_arn
}
