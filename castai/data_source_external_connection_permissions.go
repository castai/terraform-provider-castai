package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
)

const (
	FieldDSExtConnPermissionsCloudProvider   = "cloud_provider"
	FieldDSExtConnPermissionsConnectionScope  = "connection_scope"
	FieldDSExtConnPermissionsFeatures         = "features"
	FieldDSExtConnPermissionsItems            = "permissions"
	FieldDSExtConnPermissionsK8sComponents    = "kubernetes_components"

	// features block for permissions data source
	FieldDSExtConnPermFeatFeature          = "feature"
	FieldDSExtConnPermFeatRegistryVersion  = "registry_version"
	FieldDSExtConnPermFeatSubFeatures      = "sub_features"

	// permission item fields
	FieldDSExtConnPermName          = "name"
	FieldDSExtConnPermAccessType    = "access_type"
	FieldDSExtConnPermResourceType  = "resource_type"
	FieldDSExtConnPermScope         = "scope"
	FieldDSExtConnPermFeatureID     = "feature_id"
	FieldDSExtConnPermSubFeatureID  = "sub_feature_id"
	FieldDSExtConnPermJustification  = "justification"
	FieldDSExtConnPermResourceID     = "resource_id"

	// k8s component fields
	FieldDSExtConnK8sComponentID       = "component_id"
	FieldDSExtConnK8sComponentName     = "component_name"
	FieldDSExtConnK8sUsedBy            = "used_by"
	FieldDSExtConnK8sPermissions      = "permissions"
	FieldDSExtConnK8sPermApiGroup     = "api_group"
	FieldDSExtConnK8sPermNamespace     = "namespace"
	FieldDSExtConnK8sPermRbacScope     = "rbac_scope"
	FieldDSExtConnK8sPermResources     = "resources"
	FieldDSExtConnK8sPermResourceNames = "resource_names"
	FieldDSExtConnK8sPermVerbs         = "verbs"
	FieldDSExtConnK8sPermJustification = "justification"
)

func dataSourceExternalConnectionPermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceExternalConnectionPermissionsRead,
		Description: "Computes the IAM permissions and Kubernetes RBAC rules required for a given cloud provider, connection scope, and feature selection. This is a pure compute operation with no side effects — useful for previewing the permissions needed before provisioning.",
		Schema: map[string]*schema.Schema{
			FieldDSExtConnPermissionsCloudProvider: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The cloud provider to compute permissions for. Valid values: AWS, AZURE, GCP, ORACLE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS", "AZURE", "GCP", "ORACLE"}, false)),
			},
			FieldDSExtConnPermissionsConnectionScope: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Whether to include organization-level grants. Defaults to ORGANIZATION when unspecified — set to ACCOUNT to opt out of org-level grants. Valid values: AWS_ACCOUNT, AWS_ORGANIZATION, AZURE_SUBSCRIPTION, GCP_ORGANIZATION, GCP_PROJECT.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS_ACCOUNT", "AWS_ORGANIZATION", "AZURE_SUBSCRIPTION", "GCP_ORGANIZATION", "GCP_PROJECT"}, false)),
			},
			FieldDSExtConnPermissionsFeatures: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The features to compute permissions for.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldDSExtConnPermFeatFeature: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "The feature type. Valid values: CLOUD_CONNECT, COST_MONITORING, NODE_AUTOSCALING, WORKLOAD_AUTOSCALING, KARPENTER_ENTERPRISE, STORAGE_OPTIMIZATION, POD_MUTATIONS, APA, DBO, OMNI, KIMCHI.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"CLOUD_CONNECT", "COST_MONITORING", "NODE_AUTOSCALING", "WORKLOAD_AUTOSCALING", "KARPENTER_ENTERPRISE", "STORAGE_OPTIMIZATION", "POD_MUTATIONS", "APA", "DBO", "OMNI", "KIMCHI"}, false)),
						},
						FieldDSExtConnPermFeatRegistryVersion: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Registry version to pin for this feature. When set, permissions are computed for that version's snapshot. When unset, the latest version is used.",
						},
						FieldDSExtConnPermFeatSubFeatures: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Sub-features to include. Empty means base permissions only.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			FieldDSExtConnPermissionsItems: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Cloud IAM permissions the customer must grant in their cloud account.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldDSExtConnPermName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The permission name — an IAM action, role name, policy name, or policy statement depending on resource_type.",
						},
						FieldDSExtConnPermAccessType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Whether this permission is read-only or allows write/mutation. Values: READ, WRITE.",
						},
						FieldDSExtConnPermResourceType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The kind of IAM resource this permission belongs to.",
						},
						FieldDSExtConnPermScope: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Scope at which this permission must be granted.",
						},
						FieldDSExtConnPermFeatureID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The feature this permission belongs to.",
						},
						FieldDSExtConnPermSubFeatureID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The sub-feature this permission belongs to. Empty if from base permissions.",
						},
						FieldDSExtConnPermJustification: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable explanation of why this permission is needed.",
						},
						FieldDSExtConnPermResourceID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the resource (policy ID, role ID, etc.). Only set for AWS_CUSTOM_POLICY, GCP_CUSTOM_ROLE, and AZURE_CUSTOM_ROLE.",
						},
					},
				},
			},
			FieldDSExtConnPermissionsK8sComponents: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Kubernetes components and their RBAC rules to grant in the cluster.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldDSExtConnK8sComponentID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Component that requires these permissions.",
						},
						FieldDSExtConnK8sComponentName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable component name.",
						},
						FieldDSExtConnK8sUsedBy: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Features that pull in this component. Shared components appear once with multiple entries.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"feature_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Feature that pulls in the component.",
									},
									"sub_feature_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Sub-feature, if applicable. Empty for base.",
									},
								},
							},
						},
						FieldDSExtConnK8sPermissions: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "RBAC rules this component needs.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldDSExtConnK8sPermApiGroup: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "API group of the resource (empty for core API group).",
									},
									FieldDSExtConnK8sPermJustification: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Why this permission is needed.",
									},
									FieldDSExtConnK8sPermNamespace: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Namespace for namespace-scoped permissions. Empty for cluster-scoped.",
									},
									FieldDSExtConnK8sPermRbacScope: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "RBAC scope. Values: CLUSTER, NAMESPACE.",
									},
									FieldDSExtConnK8sPermResources: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Resource types (e.g. pods, deployments, nodes).",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									FieldDSExtConnK8sPermResourceNames: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Specific resource names, if scoped. Empty for all resources.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									FieldDSExtConnK8sPermVerbs: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "RBAC verbs granted (e.g. get, list, create, patch).",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceExternalConnectionPermissionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient

	req := external_connections.ComputePermissionsRequest{
		CloudProvider: external_connections.ComputePermissionsRequestCloudProvider(d.Get(FieldDSExtConnPermissionsCloudProvider).(string)),
		Features:      buildPermissionsFeatureSelections(d),
	}

	if scope, ok := d.GetOk(FieldDSExtConnPermissionsConnectionScope); ok {
		s := external_connections.ComputePermissionsRequestConnectionScope(scope.(string))
		req.ConnectionScope = &s
	}

	resp, err := client.ExternalConnectionsAPIComputePermissionsWithResponse(ctx, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.FromErr(fmt.Errorf("computing external connection permissions: %w", e))
	}

	if resp.JSON200 == nil {
		return diag.Errorf("computing external connection permissions: empty response body")
	}

	// Set permissions
	permissions := make([]map[string]any, 0)
	if resp.JSON200.Permissions != nil {
		for _, p := range *resp.JSON200.Permissions {
			m := map[string]any{
				FieldDSExtConnPermName:         p.Name,
				FieldDSExtConnPermAccessType:   string(p.AccessType),
				FieldDSExtConnPermResourceType: string(p.ResourceType),
				FieldDSExtConnPermScope:        string(p.Scope),
				FieldDSExtConnPermFeatureID:    p.FeatureId,
				FieldDSExtConnPermJustification: p.Justification,
			}
			if p.SubFeatureId != nil {
				m[FieldDSExtConnPermSubFeatureID] = *p.SubFeatureId
			} else {
				m[FieldDSExtConnPermSubFeatureID] = ""
			}
			if p.ResourceId != nil {
				m[FieldDSExtConnPermResourceID] = *p.ResourceId
			} else {
				m[FieldDSExtConnPermResourceID] = ""
			}
			permissions = append(permissions, m)
		}
	}
	if err := d.Set(FieldDSExtConnPermissionsItems, permissions); err != nil {
		return diag.FromErr(fmt.Errorf("setting permissions: %w", err))
	}

	// Set kubernetes_components
	k8sComponents := make([]map[string]any, 0)
	if resp.JSON200.KubernetesComponents != nil {
		for _, kc := range *resp.JSON200.KubernetesComponents {
			m := map[string]any{
				FieldDSExtConnK8sComponentID: kc.ComponentId,
			}
			if kc.ComponentName != nil {
				m[FieldDSExtConnK8sComponentName] = *kc.ComponentName
			} else {
				m[FieldDSExtConnK8sComponentName] = ""
			}

			// used_by
			usedBy := make([]map[string]any, 0, len(kc.UsedBy))
			for _, ub := range kc.UsedBy {
				ubMap := map[string]any{
					"feature_id": ub.FeatureId,
				}
				if ub.SubFeatureId != nil {
					ubMap["sub_feature_id"] = *ub.SubFeatureId
				} else {
					ubMap["sub_feature_id"] = ""
				}
				usedBy = append(usedBy, ubMap)
			}
			m[FieldDSExtConnK8sUsedBy] = usedBy

			// permissions (RBAC rules)
			rbacRules := make([]map[string]any, 0, len(kc.Permissions))
			for _, rule := range kc.Permissions {
				ruleMap := map[string]any{
					FieldDSExtConnK8sPermJustification: rule.Justification,
					FieldDSExtConnK8sPermResources:      toStringListAny(rule.Resources),
					FieldDSExtConnK8sPermVerbs:         toStringListAny(rule.Verbs),
				}
				if rule.ApiGroup != nil {
					ruleMap[FieldDSExtConnK8sPermApiGroup] = *rule.ApiGroup
				} else {
					ruleMap[FieldDSExtConnK8sPermApiGroup] = ""
				}
				if rule.Namespace != nil {
					ruleMap[FieldDSExtConnK8sPermNamespace] = *rule.Namespace
				} else {
					ruleMap[FieldDSExtConnK8sPermNamespace] = ""
				}
				if rule.RbacScope != nil {
					ruleMap[FieldDSExtConnK8sPermRbacScope] = string(*rule.RbacScope)
				} else {
					ruleMap[FieldDSExtConnK8sPermRbacScope] = ""
				}
				if rule.ResourceNames != nil {
					ruleMap[FieldDSExtConnK8sPermResourceNames] = toStringListAny(*rule.ResourceNames)
				} else {
					ruleMap[FieldDSExtConnK8sPermResourceNames] = []string{}
				}
				rbacRules = append(rbacRules, ruleMap)
			}
			m[FieldDSExtConnK8sPermissions] = rbacRules

			k8sComponents = append(k8sComponents, m)
		}
	}
	if err := d.Set(FieldDSExtConnPermissionsK8sComponents, k8sComponents); err != nil {
		return diag.FromErr(fmt.Errorf("setting kubernetes_components: %w", err))
	}

	d.SetId("external_connection_permissions")

	return nil
}

func buildPermissionsFeatureSelections(d *schema.ResourceData) []external_connections.FeatureSelection {
	features := d.Get(FieldDSExtConnPermissionsFeatures).([]any)
	out := make([]external_connections.FeatureSelection, 0, len(features))
	for _, f := range features {
		m := f.(map[string]any)
		fs := external_connections.FeatureSelection{
			Feature: external_connections.FeatureSelectionFeature(m[FieldDSExtConnPermFeatFeature].(string)),
		}
		if rv, ok := m[FieldDSExtConnPermFeatRegistryVersion].(string); ok && rv != "" {
			fs.RegistryVersion = toPtr(rv)
		}
		if sfs, ok := m[FieldDSExtConnPermFeatSubFeatures].([]any); ok && len(sfs) > 0 {
			subs := make([]external_connections.FeatureSelectionSubFeatures, 0, len(sfs))
			for _, sf := range sfs {
				if s, ok := sf.(string); ok && s != "" {
					subs = append(subs, external_connections.FeatureSelectionSubFeatures(s))
				}
			}
			if len(subs) > 0 {
				fs.SubFeatures = &subs
			}
		}
		out = append(out, fs)
	}
	return out
}

func toStringListAny(items []string) []string {
	return append(make([]string, 0, len(items)), items...)
}
