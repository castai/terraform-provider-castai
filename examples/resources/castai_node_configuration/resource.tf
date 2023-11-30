variable "init_script" {
  type    = string
  default = <<EOF
#!/bin/bash
echo "hello"
EOF
}

resource "castai_node_configuration" "default" {
  name           = "default"
  cluster_id     = castai_eks_cluster.test.id
  disk_cpu_ratio = 35
  min_disk_size  = 133
  subnets        = aws_subnet.test[*].id
  init_script    = base64encode(var.init_script)
  docker_config = jsonencode({
    "insecure-registries"      = ["registry.com:5000"],
    "max-concurrent-downloads" = 10
  })
  kubelet_config = jsonencode({
    "registryBurst" : 20,
    "registryPullQPS" : 10
  })
  container_runtime = "dockerd"
  tags = {
    env = "development"
  }
  eks {
    instance_profile_arn = aws_iam_instance_profile.test.arn
    dns_cluster_ip       = "10.100.0.10"
    security_groups      = [aws_security_group.test.id]
  }
}