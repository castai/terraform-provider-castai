package gke

import (
	"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("User policy", func(t *testing.T) {
		userpolicy, err := GetUserPolicy()
		if err != nil {
			t.Error(err)
		}
		if userpolicy == nil {
			t.Fatalf("couldn't generate user policy")
		}

		clustersGet := "container.clusters.get"
		zonesGet := "serviceusage.services.list"

		if !contains(userpolicy, clustersGet) || !contains(userpolicy, zonesGet) {
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
