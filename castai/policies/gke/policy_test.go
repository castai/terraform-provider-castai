package gke

import (
	"github.com/stretchr/testify/require"
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

		wantClustersGet := "container.clusters.get"
		wantZonesGet := "serviceusage.services.list"

		if !contains(userpolicy, wantClustersGet) || !contains(userpolicy, wantZonesGet) {
			t.Fatalf("generated User policy document does not contain required policies")
		}
		require.Equal(t, 38, len(userpolicy))
	})
	t.Run("LoadBalancersTargetBackendPools policy", func(t *testing.T) {
		lbTbpPolicy, err := GetLoadBalancersTargetBackendPoolsPolicy()
		if err != nil {
			t.Error(err)
		}
		if lbTbpPolicy == nil {
			t.Fatalf("couldn't generate LoadBalancersTargetBackendPools policy")
		}

		want := "compute.targetPools.get"

		if !contains(lbTbpPolicy, want) {
			t.Fatalf("generated LoadBalancersTargetBackendPools policy document does not contain required policies")
		}
		require.Equal(t, 4, len(lbTbpPolicy))
	})
	t.Run("LoadBalancersUnmanagedInstanceGroups policy", func(t *testing.T) {
		lbUigPolicy, err := GetLoadBalancersUnmanagedInstanceGroupsPolicy()
		if err != nil {
			t.Error(err)
		}
		if lbUigPolicy == nil {
			t.Fatalf("couldn't generate LoadBalancersUnmanagedInstanceGroups policy")
		}

		want := "compute.instanceGroups.update"

		if !contains(lbUigPolicy, want) {
			t.Fatalf("generated LoadBalancersUnmanagedInstanceGroups policy document does not contain required policies")
		}
		require.Equal(t, 2, len(lbUigPolicy))
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
