resource "helm_release" "nvidia_device_plugin" {
  count = var.install_nvidia_device_plugin ? 1 : 0

  name       = "nvdp"
  repository = "https://nvidia.github.io/k8s-device-plugin"
  chart      = "nvidia-device-plugin"
  namespace  = "castai-agent"

  values = [
    yamlencode({
      nodeSelector = {
        "nvidia.com/gpu" = "true"
      }

      tolerations = [
        {
          key      = "CriticalAddonsOnly"
          operator = "Exists"
        },
        {
          effect   = "NoSchedule"
          key      = "nvidia.com/gpu"
          operator = "Exists"
        },
        {
          key      = "scheduling.cast.ai/spot"
          operator = "Exists"
        },
        {
          key      = "scheduling.cast.ai/scoped-autoscaler"
          operator = "Exists"
        },
        {
          key      = "scheduling.cast.ai/node-template"
          operator = "Exists"
        },
      ]

      affinity = {
        nodeAffinity = {
          requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms = [
              {
                matchExpressions = [
                  {
                    key      = "feature.node.kubernetes.io/pci-10de.present"
                    operator = "In"
                    values   = ["true"]
                  },
                  {
                    key      = "nvidia.com/gpu.dra"
                    operator = "NotIn"
                    values   = ["true"]
                  },
                ]
              },
              {
                matchExpressions = [
                  {
                    key      = "feature.node.kubernetes.io/cpu-model.vendor_id"
                    operator = "In"
                    values   = ["NVIDIA"]
                  },
                  {
                    key      = "nvidia.com/gpu.dra"
                    operator = "NotIn"
                    values   = ["true"]
                  },
                ]
              },
              {
                matchExpressions = [
                  {
                    key      = "nvidia.com/gpu.present"
                    operator = "In"
                    values   = ["true"]
                  },
                  {
                    key      = "nvidia.com/gpu.dra"
                    operator = "NotIn"
                    values   = ["true"]
                  },
                ]
              },
            ]
          }
        }
      }
    })
  ]

  depends_on = [module.castai_eks_cluster]
}
