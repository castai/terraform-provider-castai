package policies

import (
	"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("GKE User policy", func(t *testing.T) {
		userpolicy, err := GetGKEPolicy()
		if err != nil {
			t.Error(err)
		}
		if userpolicy == nil {
			t.Fatalf("couldn't generate GKE user policy")
		}

		clustersGet := "container.clusters.get"
		zonesGet := "serviceusage.services.list"

		if !contains(userpolicy, clustersGet) || !contains(userpolicy, zonesGet) {
			t.Fatalf("generated User policy document does not contain required policies")
		}
	})

	t.Run("AKS User policy", func(t *testing.T) {
		userpolicy, err := GetAKSPolicy()
		if err != nil {
			t.Error(err)
		}
		if userpolicy == nil {
			t.Fatalf("couldn't generate AKS user policy")
		}

		computeRead := "Microsoft.Compute/*/read"
		runCommand := "Microsoft.ContainerService/managedClusters/runCommand/action"

		if !contains(userpolicy, computeRead) || !contains(userpolicy, runCommand) {
			t.Fatalf("generated User policy document does not contain required policies")
		}
	})
}

func contains(policies []string, policy string) bool {
	for _, p := range policies {
		if p == policy {
			return true
		}
	}
	return false
}
