output "grafana_password" {
  value = random_password.grafana_admin.result

  sensitive = true
}

output "ingress_ips" {
  value = [for ip in aws_eip.this : ip.public_ip]
}