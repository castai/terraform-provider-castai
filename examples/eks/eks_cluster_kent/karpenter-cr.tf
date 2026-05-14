resource "kubectl_manifest" "karpenter_default_nodeclass" {
  yaml_body = templatefile("${path.module}/karpenter-default-nodeclass.yaml.tftpl", {
    cluster_name = local.name
  })

  depends_on = [helm_release.karpenter]
}

resource "kubectl_manifest" "karpenter_default_nodepool" {
  yaml_body = file("${path.module}/karpenter-default-nodepool.yaml")

  depends_on = [
    helm_release.karpenter,
    kubectl_manifest.karpenter_default_nodeclass,
  ]
}

# Terminate Karpenter-spawned EC2 instances before the Karpenter controller
# and its CRDs go away during `tofu destroy`. Without this, helm_release.karpenter
# terminates first, NodeClaim finalizers never run, and the underlying EC2
# instances leak. Their ENIs then block VPC subnet deletion with
# `DependencyViolation`, observed across multiple destroy attempts.
#
# Two-stage approach:
#   1. Graceful — `kubectl delete nodeclaim --wait` gives Karpenter a chance
#      to drain pods and call EC2 terminate via its finalizer. Fast and clean
#      when Karpenter is healthy.
#   2. Force fallback — `aws ec2 terminate-instances` filtered by the
#      `karpenter.sh/nodepool` tag (which Karpenter writes on every instance
#      it provisions, MNG instances don't have it). Bypasses Karpenter's
#      finalizer entirely, so it works even when the controller is mid-teardown,
#      pods can't evict, or the cluster API is partially gone.
#
# Stage 1 by itself is not enough — was tried first and timed out at 5min when
# pods couldn't evict during cluster disruption, leaking 5 EC2s. Stage 2 alone
# would work but is less graceful (NodeClaim CRs left with finalizers, harmless
# since the cluster is being destroyed). Keeping both gives the best UX.
#
# `triggers` captures cluster_name + region at apply time so `self.triggers`
# can read them at destroy time — destroy-time provisioners can't dereference
# non-self resource attributes, so this is the standard pattern.
#
# Host running `tofu destroy` needs `aws` and `kubectl` CLIs in PATH and AWS
# credentials in scope. `|| true` on the graceful stage keeps destroy moving
# even if kubectl/Karpenter is unreachable; the force stage is unconditional.
resource "null_resource" "karpenter_drain" {
  triggers = {
    cluster_name = module.eks.cluster_name
    region       = var.cluster_region
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      set -euo pipefail
      KUBECONFIG_PATH="$(mktemp -t kc-${self.triggers.cluster_name}.XXXXXX)"
      trap 'rm -f "$KUBECONFIG_PATH"' EXIT

      # Stage 1: graceful drain via Karpenter.
      aws eks update-kubeconfig \
        --name "${self.triggers.cluster_name}" \
        --region "${self.triggers.region}" \
        --kubeconfig "$KUBECONFIG_PATH" || true
      KUBECONFIG="$KUBECONFIG_PATH" \
        kubectl delete nodeclaim --all --wait --timeout=2m || true

      # Stage 2: force-terminate any Karpenter-tagged instances that survived.
      IDS=$(aws ec2 describe-instances \
        --region "${self.triggers.region}" \
        --filters \
          "Name=tag:karpenter.sh/nodepool,Values=*" \
          "Name=tag:aws:eks:cluster-name,Values=${self.triggers.cluster_name}" \
          "Name=instance-state-name,Values=running,pending" \
        --query 'Reservations[].Instances[].InstanceId' \
        --output text)
      if [ -n "$IDS" ]; then
        echo "Force-terminating leaked Karpenter instances: $IDS"
        aws ec2 terminate-instances \
          --region "${self.triggers.region}" \
          --instance-ids $IDS >/dev/null
        aws ec2 wait instance-terminated \
          --region "${self.triggers.region}" \
          --instance-ids $IDS
      fi
    EOT
  }

  depends_on = [
    helm_release.karpenter,
    kubectl_manifest.karpenter_default_nodepool,
    kubectl_manifest.karpenter_default_nodeclass,
  ]
}
