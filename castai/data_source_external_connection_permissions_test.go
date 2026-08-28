package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
	mock_external_connections "github.com/castai/terraform-provider-castai/castai/sdk/external_connections/mock"
)

func TestDataSourceExternalConnectionPermissions(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := &ProviderConfig{
		externalConnectionsClient: &external_connections.ClientWithResponses{ClientInterface: mockClient},
	}

	apiGroup := ""
	namespace := ""
	rbacScope := external_connections.KubernetesPermissionRuleRbacScopeCLUSTER
	subFeatureID := ""

	permResp := external_connections.ComputePermissionsResponse{
		Permissions: &[]external_connections.Permission{
			{
				Name:         "iam.serviceAccounts.actAs",
				AccessType:   external_connections.WRITE,
				ResourceType: external_connections.GCPIAMBINDING,
				Scope:        external_connections.PermissionScopeGCPPROJECT,
				FeatureId:    "cloud_connect",
				Justification: "Allow Cast to impersonate the customer service account.",
			},
		},
		KubernetesComponents: &[]external_connections.KubernetesComponent{
			{
				ComponentId:   "castai-agent",
				ComponentName: toPtr("Cast AI Agent"),
				Permissions: []external_connections.KubernetesPermissionRule{
					{
						ApiGroup:      &apiGroup,
						Justification: "Read cluster state for autoscaling",
						Namespace:     &namespace,
						RbacScope:     &rbacScope,
						Resources:     []string{"pods", "nodes"},
						Verbs:         []string{"get", "list", "watch"},
					},
				},
				UsedBy: []external_connections.KubernetesComponentUsedBy{
					{
						FeatureId:    "node_autoscaling",
						SubFeatureId: &subFeatureID,
					},
				},
			},
		},
	}
	body, err := json.Marshal(permResp)
	r.NoError(err)

	mockClient.EXPECT().
		ExternalConnectionsAPIComputePermissions(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil)

	ds := dataSourceExternalConnectionPermissions()
	data := ds.TestResourceData()
	require.NoError(t, data.Set(FieldDSExtConnPermissionsCloudProvider, "GCP"))
	require.NoError(t, data.Set(FieldDSExtConnPermissionsConnectionScope, "GCP_PROJECT"))
	require.NoError(t, data.Set(FieldDSExtConnPermissionsFeatures, []map[string]any{
		{"feature": "CLOUD_CONNECT"},
	}))

	diags := dataSourceExternalConnectionPermissionsRead(context.Background(), data, provider)
	r.False(diags.HasError(), "read should not return errors: %v", diags)

	r.Equal("external_connection_permissions", data.Id())

	// Check permissions
	perms := data.Get(FieldDSExtConnPermissionsItems).([]any)
	r.Len(perms, 1)
	perm := perms[0].(map[string]any)
	r.Equal("iam.serviceAccounts.actAs", perm[FieldDSExtConnPermName])
	r.Equal("WRITE", perm[FieldDSExtConnPermAccessType])
	r.Equal("GCP_IAM_BINDING", perm[FieldDSExtConnPermResourceType])
	r.Equal("GCP_PROJECT", perm[FieldDSExtConnPermScope])
	r.Equal("cloud_connect", perm[FieldDSExtConnPermFeatureID])

	// Check kubernetes_components
	k8sComps := data.Get(FieldDSExtConnPermissionsK8sComponents).([]any)
	r.Len(k8sComps, 1)
	kc := k8sComps[0].(map[string]any)
	r.Equal("castai-agent", kc[FieldDSExtConnK8sComponentID])
	r.Equal("Cast AI Agent", kc[FieldDSExtConnK8sComponentName])

	// Check used_by
	usedBy := kc[FieldDSExtConnK8sUsedBy].([]any)
	r.Len(usedBy, 1)
	ub := usedBy[0].(map[string]any)
	r.Equal("node_autoscaling", ub["feature_id"])

	// Check RBAC rules
	rbacRules := kc[FieldDSExtConnK8sPermissions].([]any)
	r.Len(rbacRules, 1)
	rule := rbacRules[0].(map[string]any)
	r.Equal("Read cluster state for autoscaling", rule[FieldDSExtConnK8sPermJustification])
	r.Equal("CLUSTER", rule[FieldDSExtConnK8sPermRbacScope])

	resources := rule[FieldDSExtConnK8sPermResources].([]any)
	r.Len(resources, 2)
	r.Equal("pods", resources[0])
	r.Equal("nodes", resources[1])

	verbs := rule[FieldDSExtConnK8sPermVerbs].([]any)
	r.Len(verbs, 3)
	r.Equal("get", verbs[0])
	r.Equal("list", verbs[1])
	r.Equal("watch", verbs[2])
}
