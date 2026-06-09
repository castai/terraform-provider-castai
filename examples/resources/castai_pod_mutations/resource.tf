# castai-pod-mutator helm chart installs Kubernetes Custom Resource Definition (CRD) for the kind 'PodMutation'.
# Pod mutation rules can then be added to a cluster as plain Kubernetes object of this kind.
#
# You should use a name that is _not_ shared by a mutation created via Cast AI console because.
# In such cases, custom resource mutation will be shadowed by the mutation created via the console.

resource "kubernetes_manifest" "test_pod_mutation" {
  manifest = {
    apiVersion = "pod-mutations.cast.ai/v1"
    kind       = "PodMutation"
    metadata = {
      name = "test-pod-mutation"
    }
    spec = {
      filter = {
        # Filter values can be plain strings of regexes.
        workload = {
          namespaces = ["production", "staging"]
          names      = ["^frontend-.*$", "^backend-.*$"]
          kinds      = ["Pod", "Deployment", "ReplicaSet"]
        }
        pod = {
          # labelsOperator can be "and" or "or"
          labelsOperator = "and"
          labelsFilter = [
            {
              key   = "app.kubernetes.io/part-of"
              value = "platform"
            },
            {
              key   = "tier"
              value = "frontend"
            }
          ]
        }
      }
      restartPolicy = "deferred"
      patches = [
        {
          op    = "add"
          path  = "/metadata/annotations/mutated-by-pod-mutator"
          value = "true"
        }
      ]
      spotConfig = {
        # mode can be "preferred-spot", "optional-spot", or "only-spot"
        mode                   = "preferred-spot"
        distributionPercentage = 50
      }
    }
  }
}
