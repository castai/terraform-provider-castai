package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/patching_engine"
)

const (
	FieldPodMutationOrganizationID           = "organization_id"
	FieldPodMutationClusterID                = "cluster_id"
	FieldPodMutationName                     = "name"
	FieldPodMutationEnabled                  = "enabled"
	FieldPodMutationFilterV2                 = "filter_v2"
	FieldPodMutationFilterWorkload           = "workload"
	FieldPodMutationFilterPod                = "pod"
	FieldPodMutationLabels                   = "labels"
	FieldPodMutationAnnotations              = "annotations"
	FieldPodMutationNodeSelector             = "node_selector"
	FieldPodMutationNodeSelectorAdd          = "add"
	FieldPodMutationNodeSelectorRemove       = "remove"
	FieldPodMutationTolerations              = "tolerations"
	FieldPodMutationTolerationKey            = "key"
	FieldPodMutationTolerationOperator       = "operator"
	FieldPodMutationTolerationValue          = "value"
	FieldPodMutationTolerationEffect         = "effect"
	FieldPodMutationTolerationSeconds        = "toleration_seconds"
	FieldPodMutationAffinity                 = "affinity"
	FieldPodMutationNodeAffinity             = "node_affinity"
	FieldPodMutationPreferred                = "preferred_during_scheduling_ignored_during_execution"
	FieldPodMutationWeight                   = "weight"
	FieldPodMutationPreference               = "preference"
	FieldPodMutationMatchExpressions         = "match_expressions"
	FieldPodMutationMatchExpressionsKey      = "key"
	FieldPodMutationMatchExpressionsOperator = "operator"
	FieldPodMutationSpotConfig               = "spot_config"
	FieldPodMutationSpotMode                 = "spot_mode"
	FieldPodMutationSpotType                 = "spot_type"
	FieldPodMutationSpotDistributionPct      = "distribution_percentage"
	FieldPodMutationNodeTemplates            = "node_templates_to_consolidate"
	FieldPodMutationPatch                    = "patch"
	FieldPodMutationDistributionGroups       = "distribution_groups"
	FieldPodMutationDistributionGroupName    = "name"
	FieldPodMutationDistributionGroupPct     = "percentage"
	FieldPodMutationDistributionGroupConfiguration = "configuration"
	FieldPodMutationSource                   = "source"
	FieldPodMutationFilterNames              = "names"
	FieldPodMutationFilterNamespaces         = "namespaces"
	FieldPodMutationFilterKinds              = "kinds"
	FieldPodMutationFilterLabelsFilter       = "labels_filter"
	FieldPodMutationFilterExcludeNames       = "exclude_names"
	FieldPodMutationFilterExcludeNamespaces  = "exclude_namespaces"
	FieldPodMutationFilterExcludeKinds       = "exclude_kinds"
	FieldPodMutationFilterExcludeLabels      = "exclude_labels_filter"
	FieldPodMutationMatcherType              = "type"
	FieldPodMutationMatcherValue             = "value"
	FieldPodMutationLabelsFilterOperator     = "operator"
	FieldPodMutationLabelsFilterMatchers     = "matchers"
	FieldPodMutationLabelMatcherKey          = "key"
	FieldPodMutationLabelMatcherValue        = "value"
	FieldPodMutationValues                   = "values"
)

var spotModeValues = []string{
	string(patching_engine.PodMutationSpotTypeOPTIONALSPOT),
	string(patching_engine.PodMutationSpotTypePREFERREDSPOT),
	string(patching_engine.PodMutationSpotTypeUSEONLYSPOT),
}

var matcherTypeValues = []string{
	string(patching_engine.EXACT),
	string(patching_engine.REGEX),
}

var labelsFilterOperatorValues = []string{
	string(patching_engine.AND),
	string(patching_engine.OR),
}

var matcherSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		FieldPodMutationMatcherType: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(matcherTypeValues, false)),
			Description:      "Matcher type: EXACT or REGEX.",
		},
		FieldPodMutationMatcherValue: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Value to match against.",
		},
	},
}

var labelsFilterSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		FieldPodMutationLabelsFilterOperator: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(labelsFilterOperatorValues, false)),
			Description:      "Logical operator to combine label matchers: AND or OR.",
		},
		FieldPodMutationLabelsFilterMatchers: {
			Type:     schema.TypeList,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					FieldPodMutationLabelMatcherKey: {
						Type:     schema.TypeList,
						Required: true,
						MaxItems: 1,
						Elem:     matcherSchema,
					},
					FieldPodMutationLabelMatcherValue: {
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem:     matcherSchema,
					},
				},
			},
		},
	},
}

var tolerationSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		FieldPodMutationTolerationKey: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Toleration key.",
		},
		FieldPodMutationTolerationOperator: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Toleration operator.",
		},
		FieldPodMutationTolerationValue: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Toleration value.",
		},
		FieldPodMutationTolerationEffect: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Toleration effect.",
		},
		FieldPodMutationTolerationSeconds: {
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "Toleration seconds.",
		},
	},
}

var nodeSelectorSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		FieldPodMutationNodeSelectorAdd: {
			Type:     schema.TypeMap,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		FieldPodMutationNodeSelectorRemove: {
			Type:     schema.TypeMap,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
	},
}

var affinitySchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		FieldPodMutationNodeAffinity: {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					FieldPodMutationPreferred: {
						Type:     schema.TypeList,
						Optional: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								FieldPodMutationWeight: {
									Type:        schema.TypeInt,
									Required:    true,
									Description: "Weight of the node affinity term.",
								},
								FieldPodMutationPreference: {
									Type:     schema.TypeList,
									Required: true,
									MaxItems: 1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											FieldPodMutationMatchExpressions: {
												Type:     schema.TypeList,
												Optional: true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														FieldPodMutationMatchExpressionsKey: {
															Type:     schema.TypeString,
															Required: true,
														},
														FieldPodMutationMatchExpressionsOperator: {
															Type:     schema.TypeString,
															Required: true,
														},
														FieldPodMutationValues: {
															Type:     schema.TypeList,
															Optional: true,
															Elem:     &schema.Schema{Type: schema.TypeString},
														},
													},
												},
											},
										},
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

var mutationConfigSchema = map[string]*schema.Schema{
	FieldPodMutationLabels: {
		Type:        schema.TypeMap,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Labels to add to the pods.",
	},
	FieldPodMutationAnnotations: {
		Type:        schema.TypeMap,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Annotations to add to the pods.",
	},
	FieldPodMutationNodeSelector: {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Elem:        nodeSelectorSchema,
		Description: "Node selector to apply to the pods (add/remove key-value pairs).",
	},
	FieldPodMutationTolerations: {
		Type:        schema.TypeList,
		Optional:    true,
		Elem:        tolerationSchema,
		Description: "Tolerations to apply to the pods.",
	},
	FieldPodMutationAffinity: {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Elem:        affinitySchema,
		Description: "Affinity to apply to the pods.",
	},
	FieldPodMutationSpotType: {
		Type:             schema.TypeString,
		Optional:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(spotModeValues, false)),
		Description:      "Spot instance type: OPTIONAL_SPOT, USE_ONLY_SPOT, or PREFERRED_SPOT.",
	},
	FieldPodMutationNodeTemplates: {
		Type:        schema.TypeList,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Node template names to consolidate.",
	},
	FieldPodMutationPatch: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "JSON patch to apply to pods. Must be a JSON array of patch operations.",
		ValidateDiagFunc: validation.ToDiagFunc(func(val interface{}, key string) ([]string, []error) {
			s := val.(string)
			if s == "" {
				return nil, nil
			}
			var arr []map[string]interface{}
			if err := json.Unmarshal([]byte(s), &arr); err != nil {
				return nil, []error{fmt.Errorf("%q must be a valid JSON array of patch operations: %w", key, err)}
			}
			validOps := map[string]struct{}{"add": {}, "remove": {}, "replace": {}, "move": {}, "copy": {}, "test": {}}
			for i, op := range arr {
				opVal, ok := op["op"].(string)
				if !ok || opVal == "" {
					return nil, []error{fmt.Errorf("%q operation %d: missing or invalid \"op\" field", key, i)}
				}
				if _, ok := validOps[opVal]; !ok {
					return nil, []error{fmt.Errorf("%q operation %d: \"op\" must be one of add, remove, replace, move, copy, test; got %q", key, i, opVal)}
				}
				path, ok := op["path"].(string)
				if !ok || path == "" {
					return nil, []error{fmt.Errorf("%q operation %d: missing or empty \"path\" field", key, i)}
				}
			}
			return nil, nil
		}),
	},
}

func resourcePodMutation() *schema.Resource {
	s := map[string]*schema.Schema{
		FieldPodMutationOrganizationID: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "ID of the organization. If not provided, will be inferred from the API client.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
		},
		FieldPodMutationClusterID: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      "ID of the cluster.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
		},
		FieldPodMutationName: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      "Name of the pod mutation.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
		},
		FieldPodMutationEnabled: {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Whether the pod mutation is enabled.",
		},
		FieldPodMutationFilterV2: {
			Type:        schema.TypeList,
			Required:    true,
			MaxItems:    1,
			Description: "Advanced object filter with support for exact and regex matching.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					FieldPodMutationFilterWorkload: {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Workload filter for kinds, names, and namespaces.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								FieldPodMutationFilterNames: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
								FieldPodMutationFilterNamespaces: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
								FieldPodMutationFilterKinds: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
								FieldPodMutationFilterExcludeNames: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
								FieldPodMutationFilterExcludeNamespaces: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
								FieldPodMutationFilterExcludeKinds: {
									Type:     schema.TypeList,
									Optional: true,
									Elem:     matcherSchema,
								},
							},
						},
					},
					FieldPodMutationFilterPod: {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Pod filter for labels.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								FieldPodMutationFilterLabelsFilter: {
									Type:     schema.TypeList,
									Optional: true,
									MaxItems: 1,
									Elem:     labelsFilterSchema,
								},
								FieldPodMutationFilterExcludeLabels: {
									Type:     schema.TypeList,
									Optional: true,
									MaxItems: 1,
									Elem:     labelsFilterSchema,
								},
							},
						},
					},
				},
			},
		},
		FieldPodMutationSpotConfig: {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Spot configuration for the mutation.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					FieldPodMutationSpotMode: {
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(spotModeValues, false)),
						Description:      "Spot mode: OPTIONAL_SPOT, USE_ONLY_SPOT, or PREFERRED_SPOT.",
					},
					FieldPodMutationSpotDistributionPct: {
						Type:             schema.TypeInt,
						Optional:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
						Description:      "Percentage of pods (0-100) that receive spot scheduling constraints.",
					},
				},
			},
		},
		FieldPodMutationDistributionGroups: {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Distribution groups for percentage-based pod distribution.",
			Elem: &schema.Resource{
				Schema: distributionGroupSchema(),
			},
		},
		FieldPodMutationSource: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Source of the pod mutation (API or CUSTOM_RESOURCE).",
		},
	}

	// Add shared mutation config fields to the top-level schema
	for k, v := range mutationConfigSchema {
		s[k] = v
	}

	return &schema.Resource{
		CreateContext: resourcePodMutationCreate,
		ReadContext:   resourcePodMutationRead,
		UpdateContext: resourcePodMutationUpdate,
		DeleteContext: resourcePodMutationDelete,

		Importer: &schema.ResourceImporter{
			StateContext: podMutationStateImporter,
		},

		CustomizeDiff: resourcePodMutationCustomizeDiff,

		Description: "CAST AI pod mutation resource allows managing pod mutations for Kubernetes workloads.",
		Schema:      s,
	}
}

func resourcePodMutationCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	filterV2 := d.Get(FieldPodMutationFilterV2).([]interface{})
	if len(filterV2) == 0 || filterV2[0] == nil {
		return fmt.Errorf("filter_v2 must not be empty")
	}
	fm := filterV2[0].(map[string]interface{})

	workloadFields := []string{
		FieldPodMutationFilterNames,
		FieldPodMutationFilterNamespaces,
		FieldPodMutationFilterKinds,
		FieldPodMutationFilterExcludeNames,
		FieldPodMutationFilterExcludeNamespaces,
		FieldPodMutationFilterExcludeKinds,
	}
	podFields := []string{
		FieldPodMutationFilterLabelsFilter,
		FieldPodMutationFilterExcludeLabels,
	}

	workloadHasFilter := false
	if wl := fm[FieldPodMutationFilterWorkload].([]interface{}); len(wl) > 0 && wl[0] != nil {
		wm := wl[0].(map[string]interface{})
		for _, f := range workloadFields {
			if v, ok := wm[f].([]interface{}); ok && len(v) > 0 {
				workloadHasFilter = true
				break
			}
		}
	}

	podHasFilter := false
	if pl := fm[FieldPodMutationFilterPod].([]interface{}); len(pl) > 0 && pl[0] != nil {
		pm := pl[0].(map[string]interface{})
		for _, f := range podFields {
			if v, ok := pm[f].([]interface{}); ok && len(v) > 0 {
				podHasFilter = true
				break
			}
		}
	}

	if !workloadHasFilter && !podHasFilter {
		return fmt.Errorf("filter_v2 must specify at least one filter in workload or pod")
	}
	return nil
}

func distributionGroupSchema() map[string]*schema.Schema {
	configSchema := map[string]*schema.Schema{}
	for k, v := range mutationConfigSchema {
		configSchema[k] = v
	}

	return map[string]*schema.Schema{
		FieldPodMutationDistributionGroupName: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			Description:      "Unique name for this distribution group.",
		},
		FieldPodMutationDistributionGroupPct: {
			Type:             schema.TypeInt,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
			Description:      "Percentage of pods (0-100) that should receive this configuration.",
		},
		FieldPodMutationDistributionGroupConfiguration: {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: configSchema,
			},
		},
	}
}

func podMutationStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid import ID %s, expected format: organization_id/cluster_id/mutation_id", d.Id())
	}
	if err := d.Set(FieldPodMutationOrganizationID, parts[0]); err != nil {
		return nil, err
	}
	if err := d.Set(FieldPodMutationClusterID, parts[1]); err != nil {
		return nil, err
	}
	d.SetId(parts[2])
	return []*schema.ResourceData{d}, nil
}

func resourcePodMutationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).patchingEngineClient

	organizationID, err := getPodMutationOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(FieldPodMutationClusterID).(string)

	body := stateToPodMutation(d)

	tflog.Info(ctx, "creating pod mutation", map[string]interface{}{
		"name":            lo.FromPtr(body.Name),
		"cluster_id":      clusterID,
		"organization_id": organizationID,
	})

	resp, err := client.PodMutationsAPICreatePodMutationWithResponse(ctx, organizationID, clusterID, body)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.Errorf("creating pod mutation: %v", checkErr)
	}

	d.SetId(lo.FromPtr(resp.JSON200.Id))

	return resourcePodMutationRead(ctx, d, meta)
}

func resourcePodMutationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).patchingEngineClient

	organizationID, err := getPodMutationOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(FieldPodMutationClusterID).(string)

	resp, err := client.PodMutationsAPIGetPodMutationWithResponse(ctx, organizationID, clusterID, d.Id())
	if err != nil {
		return diag.Errorf("getting pod mutation: %v", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "pod mutation not found, removing from state", map[string]interface{}{
			"resource_id": d.Id(),
		})
		d.SetId("")
		return nil
	}
	if checkErr := sdk.CheckOKResponse(resp, nil); checkErr != nil {
		return diag.Errorf("getting pod mutation: %v", checkErr)
	}

	return podMutationToState(resp.JSON200, d)
}

func resourcePodMutationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).patchingEngineClient

	organizationID, err := getPodMutationOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(FieldPodMutationClusterID).(string)

	body := stateToPodMutation(d)
	id := d.Id()
	body.Id = &id

	tflog.Info(ctx, "updating pod mutation", map[string]interface{}{
		"resource_id":     d.Id(),
		"cluster_id":      clusterID,
		"organization_id": organizationID,
	})

	resp, err := client.PodMutationsAPIUpdatePodMutationWithResponse(ctx, organizationID, clusterID, d.Id(), body)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.Errorf("updating pod mutation: %v", checkErr)
	}

	return resourcePodMutationRead(ctx, d, meta)
}

func resourcePodMutationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).patchingEngineClient

	organizationID, err := getPodMutationOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(FieldPodMutationClusterID).(string)

	tflog.Info(ctx, "deleting pod mutation", map[string]interface{}{
		"resource_id":     d.Id(),
		"cluster_id":      clusterID,
		"organization_id": organizationID,
	})

	resp, err := client.PodMutationsAPIDeletePodMutationWithResponse(ctx, organizationID, clusterID, d.Id())
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.Errorf("deleting pod mutation: %v", checkErr)
	}

	d.SetId("")
	return nil
}

func getPodMutationOrganizationID(ctx context.Context, d *schema.ResourceData, meta interface{}) (string, error) {
	organizationID := d.Get(FieldPodMutationOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return "", fmt.Errorf("getting organization ID: %w", err)
		}
	}
	return organizationID, nil
}

// stateToPodMutation converts Terraform state to the API request body.
func stateToPodMutation(d *schema.ResourceData) patching_engine.PodMutation {
	name := d.Get(FieldPodMutationName).(string)
	enabled := d.Get(FieldPodMutationEnabled).(bool)

	mutation := patching_engine.PodMutation{
		Name:    &name,
		Enabled: &enabled,
	}

	// Filter V2
	filterList := d.Get(FieldPodMutationFilterV2).([]interface{})
	mutation.ObjectFilterV2 = stateToObjectFilterV2(filterList[0].(map[string]interface{}))

	// Mutation config fields
	configMap := map[string]interface{}{}
	for _, key := range []string{
		FieldPodMutationLabels,
		FieldPodMutationAnnotations,
		FieldPodMutationNodeSelector,
		FieldPodMutationTolerations,
		FieldPodMutationAffinity,
		FieldPodMutationSpotType,
		FieldPodMutationNodeTemplates,
		FieldPodMutationPatch,
	} {
		configMap[key] = d.Get(key)
	}
	cfg := parseMutationConfigFromMap(configMap)
	mutation.Labels = cfg.Labels
	mutation.Annotations = cfg.Annotations
	mutation.NodeSelector = cfg.NodeSelector
	mutation.Tolerations = cfg.Tolerations
	mutation.Affinity = cfg.Affinity
	if cfg.SpotType != "" {
		st := patching_engine.PodMutationSpotType(cfg.SpotType)
		mutation.SpotType = &st
	}
	mutation.NodeTemplatesToConsolidate = cfg.NodeTemplates
	mutation.Patch = cfg.Patch

	// Spot config
	if spotList, ok := d.Get(FieldPodMutationSpotConfig).([]interface{}); ok && len(spotList) > 0 && spotList[0] != nil {
		sm := spotList[0].(map[string]interface{})
		if mode, ok := sm[FieldPodMutationSpotMode].(string); ok && mode != "" {
			st := patching_engine.PodMutationSpotType(mode)
			mutation.SpotType = &st
		}
		pct := int32(sm[FieldPodMutationSpotDistributionPct].(int))
		mutation.SpotDistributionPercentage = &pct
	}

	// Distribution groups
	groups := stateToDistributionGroups(d.Get(FieldPodMutationDistributionGroups).([]interface{}))
	mutation.DistributionGroups = &groups

	return mutation
}

// mutationConfigResult holds the parsed mutation config fields extracted from a map.
type mutationConfigResult struct {
	Labels        *map[string]string
	Annotations   *map[string]string
	NodeSelector  *patching_engine.PatchOptions
	Tolerations   *[]patching_engine.Toleration
	Affinity      *patching_engine.Affinity
	SpotType      string
	NodeTemplates *[]string
	Patch         *[]map[string]interface{}
}

// parseMutationConfigFromMap extracts mutation config fields from a raw map.
func parseMutationConfigFromMap(m map[string]interface{}) mutationConfigResult {
	var result mutationConfigResult

	if v, ok := m[FieldPodMutationLabels]; ok && v != nil {
		if labelsMap, ok := v.(map[string]interface{}); ok && len(labelsMap) > 0 {
			sm := toStringMap(labelsMap)
			result.Labels = &sm
		}
	}

	if v, ok := m[FieldPodMutationAnnotations]; ok && v != nil {
		if annotationsMap, ok := v.(map[string]interface{}); ok && len(annotationsMap) > 0 {
			sm := toStringMap(annotationsMap)
			result.Annotations = &sm
		}
	}

	if v, ok := m[FieldPodMutationNodeSelector]; ok {
		nsList := v.([]interface{})
		if len(nsList) > 0 && nsList[0] != nil {
			result.NodeSelector = stateToNodeSelector(nsList[0].(map[string]interface{}))
		}
	}

	if v, ok := m[FieldPodMutationTolerations]; ok {
		tolList := v.([]interface{})
		if len(tolList) > 0 {
			tols := stateToTolerations(tolList)
			result.Tolerations = &tols
		}
	}

	if v, ok := m[FieldPodMutationAffinity]; ok {
		affList := v.([]interface{})
		if len(affList) > 0 && affList[0] != nil {
			result.Affinity = stateToAffinity(affList[0].(map[string]interface{}))
		}
	}

	if v, ok := m[FieldPodMutationSpotType]; ok {
		if s, ok := v.(string); ok && s != "" {
			result.SpotType = s
		}
	}

	if v, ok := m[FieldPodMutationNodeTemplates]; ok {
		templates := toStringList(v.([]interface{}))
		if len(templates) > 0 {
			result.NodeTemplates = &templates
		}
	}

	if v, ok := m[FieldPodMutationPatch]; ok {
		if patchStr, ok := v.(string); ok && patchStr != "" {
			var patchArr []map[string]interface{}
			if err := json.Unmarshal([]byte(patchStr), &patchArr); err == nil {
				result.Patch = &patchArr
			}
		}
	}

	return result
}

func stateToObjectFilterV2(m map[string]interface{}) *patching_engine.ObjectFilterV2 {
	filter := &patching_engine.ObjectFilterV2{}

	// Workload filter: kinds, names, namespaces
	if wl, ok := m[FieldPodMutationFilterWorkload]; ok {
		wlList := wl.([]interface{})
		if len(wlList) > 0 && wlList[0] != nil {
			wm := wlList[0].(map[string]interface{})
			if v, ok := wm[FieldPodMutationFilterNames]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.Names = &matchers
				}
			}
			if v, ok := wm[FieldPodMutationFilterNamespaces]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.Namespaces = &matchers
				}
			}
			if v, ok := wm[FieldPodMutationFilterKinds]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.Kinds = &matchers
				}
			}
			if v, ok := wm[FieldPodMutationFilterExcludeNames]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.ExcludeNames = &matchers
				}
			}
			if v, ok := wm[FieldPodMutationFilterExcludeNamespaces]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.ExcludeNamespaces = &matchers
				}
			}
			if v, ok := wm[FieldPodMutationFilterExcludeKinds]; ok {
				matchers := stateToMatchers(v.([]interface{}))
				if len(matchers) > 0 {
					filter.ExcludeKinds = &matchers
				}
			}
		}
	}

	// Pod filter: labels
	if p, ok := m[FieldPodMutationFilterPod]; ok {
		pList := p.([]interface{})
		if len(pList) > 0 && pList[0] != nil {
			pm := pList[0].(map[string]interface{})
			if v, ok := pm[FieldPodMutationFilterLabelsFilter]; ok {
				filterList := v.([]interface{})
				if len(filterList) > 0 && filterList[0] != nil {
					filter.Labels = stateToLabelsFilter(filterList[0].(map[string]interface{}))
				}
			}
			if v, ok := pm[FieldPodMutationFilterExcludeLabels]; ok {
				filterList := v.([]interface{})
				if len(filterList) > 0 && filterList[0] != nil {
					filter.ExcludeLabels = stateToLabelsFilter(filterList[0].(map[string]interface{}))
				}
			}
		}
	}

	return filter
}

func stateToMatchers(items []interface{}) []patching_engine.ObjectFilterV2Matcher {
	matchers := make([]patching_engine.ObjectFilterV2Matcher, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		m := item.(map[string]interface{})
		matcherType := patching_engine.ObjectFilterV2MatcherType(m[FieldPodMutationMatcherType].(string))
		value := m[FieldPodMutationMatcherValue].(string)
		matchers = append(matchers, patching_engine.ObjectFilterV2Matcher{
			Type:  &matcherType,
			Value: &value,
		})
	}
	return matchers
}

func stateToLabelsFilter(m map[string]interface{}) *patching_engine.ObjectFilterV2LabelsFilter {
	op := patching_engine.ObjectFilterV2LabelsFilterOperator(m[FieldPodMutationLabelsFilterOperator].(string))
	filter := &patching_engine.ObjectFilterV2LabelsFilter{
		Operator: &op,
	}

	if v, ok := m[FieldPodMutationLabelsFilterMatchers]; ok {
		matchersList := v.([]interface{})
		matchers := make([]patching_engine.ObjectFilterV2LabelMatcher, 0, len(matchersList))
		for _, item := range matchersList {
			if item == nil {
				continue
			}
			lm := item.(map[string]interface{})
			labelMatcher := patching_engine.ObjectFilterV2LabelMatcher{}

			if keyList, ok := lm[FieldPodMutationLabelMatcherKey]; ok {
				kl := keyList.([]interface{})
				if len(kl) > 0 && kl[0] != nil {
					km := kl[0].(map[string]interface{})
					keyType := patching_engine.ObjectFilterV2MatcherType(km[FieldPodMutationMatcherType].(string))
					keyValue := km[FieldPodMutationMatcherValue].(string)
					labelMatcher.Key = &patching_engine.ObjectFilterV2Matcher{
						Type:  &keyType,
						Value: &keyValue,
					}
				}
			}

			if valList, ok := lm[FieldPodMutationLabelMatcherValue]; ok {
				vl := valList.([]interface{})
				if len(vl) > 0 && vl[0] != nil {
					vm := vl[0].(map[string]interface{})
					valType := patching_engine.ObjectFilterV2MatcherType(vm[FieldPodMutationMatcherType].(string))
					valValue := vm[FieldPodMutationMatcherValue].(string)
					labelMatcher.Value = &patching_engine.ObjectFilterV2Matcher{
						Type:  &valType,
						Value: &valValue,
					}
				}
			}

			matchers = append(matchers, labelMatcher)
		}
		filter.Matchers = &matchers
	}

	return filter
}

func stateToNodeSelector(m map[string]interface{}) *patching_engine.PatchOptions {
	ns := &patching_engine.PatchOptions{}
	if v, ok := m[FieldPodMutationNodeSelectorAdd]; ok {
		addMap := toStringMap(v.(map[string]interface{}))
		if len(addMap) > 0 {
			ns.Add = &addMap
		}
	}
	if v, ok := m[FieldPodMutationNodeSelectorRemove]; ok {
		removeMap := toStringMap(v.(map[string]interface{}))
		if len(removeMap) > 0 {
			ns.Remove = &removeMap
		}
	}
	return ns
}

func stateToTolerations(items []interface{}) []patching_engine.Toleration {
	tolerations := make([]patching_engine.Toleration, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		m := item.(map[string]interface{})
		t := patching_engine.Toleration{}
		if v := m[FieldPodMutationTolerationKey].(string); v != "" {
			t.Key = &v
		}
		if v := m[FieldPodMutationTolerationOperator].(string); v != "" {
			t.Operator = &v
		}
		if v := m[FieldPodMutationTolerationValue].(string); v != "" {
			t.Value = &v
		}
		if v := m[FieldPodMutationTolerationEffect].(string); v != "" {
			t.Effect = &v
		}
		if v, ok := m[FieldPodMutationTolerationSeconds].(int); ok && v != 0 {
			s := strconv.Itoa(v)
			t.TolerationSeconds = &s
		}
		tolerations = append(tolerations, t)
	}
	return tolerations
}

func stateToAffinity(m map[string]interface{}) *patching_engine.Affinity {
	affinity := &patching_engine.Affinity{}

	if v, ok := m[FieldPodMutationNodeAffinity]; ok {
		naList := v.([]interface{})
		if len(naList) > 0 && naList[0] != nil {
			naMap := naList[0].(map[string]interface{})
			nodeAffinity := &patching_engine.NodeAffinity{}

			if preferred, ok := naMap[FieldPodMutationPreferred]; ok {
				prefList := preferred.([]interface{})
				terms := make([]patching_engine.NodeAffinityWeightedNodeAffinityTerm, 0, len(prefList))
				for _, p := range prefList {
					if p == nil {
						continue
					}
					pm := p.(map[string]interface{})
					weight := int32(pm[FieldPodMutationWeight].(int))
					term := patching_engine.NodeAffinityWeightedNodeAffinityTerm{
						Weight: &weight,
					}

					if prefNode, ok := pm[FieldPodMutationPreference]; ok {
						prefNodeList := prefNode.([]interface{})
						if len(prefNodeList) > 0 && prefNodeList[0] != nil {
							prefNodeMap := prefNodeList[0].(map[string]interface{})
							selectorTerm := &patching_engine.NodeSelectorTerm{}
							if exprs, ok := prefNodeMap[FieldPodMutationMatchExpressions]; ok {
								exprList := exprs.([]interface{})
								reqs := make([]patching_engine.NodeSelectorRequirement, 0, len(exprList))
								for _, e := range exprList {
									if e == nil {
										continue
									}
									em := e.(map[string]interface{})
									key := em[FieldPodMutationMatchExpressionsKey].(string)
									operator := em[FieldPodMutationMatchExpressionsOperator].(string)
									req := patching_engine.NodeSelectorRequirement{
										Key:      &key,
										Operator: &operator,
									}
									if vals, ok := em[FieldPodMutationValues]; ok {
										valStrs := toStringList(vals.([]interface{}))
										if len(valStrs) > 0 {
											req.Values = &valStrs
										}
									}
									reqs = append(reqs, req)
								}
								selectorTerm.MatchExpressions = &reqs
							}
							term.Preference = selectorTerm
						}
					}

					terms = append(terms, term)
				}
				nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = &terms
			}

			affinity.NodeAffinity = nodeAffinity
		}
	}

	return affinity
}

func stateToDistributionGroups(items []interface{}) []patching_engine.DistributionGroup {
	groups := make([]patching_engine.DistributionGroup, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		m := item.(map[string]interface{})
		name := m[FieldPodMutationDistributionGroupName].(string)
		pct := int32(m[FieldPodMutationDistributionGroupPct].(int))
		group := patching_engine.DistributionGroup{
			Name:       &name,
			Percentage: &pct,
		}

		if configList, ok := m[FieldPodMutationDistributionGroupConfiguration]; ok {
			cl := configList.([]interface{})
			if len(cl) > 0 && cl[0] != nil {
				configMap := cl[0].(map[string]interface{})
				cfg := parseMutationConfigFromMap(configMap)
				config := &patching_engine.DistributionGroupConfig{
					Labels:                     cfg.Labels,
					Annotations:                cfg.Annotations,
					NodeSelector:               cfg.NodeSelector,
					Tolerations:                cfg.Tolerations,
					Affinity:                   cfg.Affinity,
					NodeTemplatesToConsolidate: cfg.NodeTemplates,
					Patch:                      cfg.Patch,
				}
				if cfg.SpotType != "" {
					st := patching_engine.DistributionGroupConfigSpotType(cfg.SpotType)
					config.SpotType = &st
				}

				group.Config = config
			}
		}

		groups = append(groups, group)
	}
	return groups
}

// podMutationToState converts the API response to Terraform state.
func podMutationToState(mutation *patching_engine.PodMutation, d *schema.ResourceData) diag.Diagnostics {
	d.SetId(lo.FromPtr(mutation.Id))

	if err := d.Set(FieldPodMutationName, lo.FromPtr(mutation.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(FieldPodMutationEnabled, lo.FromPtr(mutation.Enabled)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(FieldPodMutationClusterID, lo.FromPtr(mutation.ClusterId)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(FieldPodMutationOrganizationID, lo.FromPtr(mutation.OrganizationId)); err != nil {
		return diag.FromErr(err)
	}

	if mutation.ObjectFilterV2 != nil {
		if err := d.Set(FieldPodMutationFilterV2, flattenObjectFilterV2(mutation.ObjectFilterV2)); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.Labels != nil {
		if err := d.Set(FieldPodMutationLabels, *mutation.Labels); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(FieldPodMutationLabels, map[string]string{}); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.Annotations != nil {
		if err := d.Set(FieldPodMutationAnnotations, *mutation.Annotations); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(FieldPodMutationAnnotations, map[string]string{}); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.NodeSelector != nil {
		if err := d.Set(FieldPodMutationNodeSelector, flattenPodMutationNodeSelector(mutation.NodeSelector)); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(FieldPodMutationNodeSelector, []map[string]interface{}{}); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.Tolerations != nil {
		flatTolerations, err := flattenTolerations(*mutation.Tolerations)
		if err != nil {
			return diag.Errorf("flattening tolerations: %v", err)
		}

		if err := d.Set(FieldPodMutationTolerations, flatTolerations); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(FieldPodMutationTolerations, []map[string]interface{}{}); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.Affinity != nil {
		if err := d.Set(FieldPodMutationAffinity, flattenAffinity(mutation.Affinity)); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(FieldPodMutationAffinity, []map[string]interface{}{}); err != nil {
			return diag.FromErr(err)
		}
	}

	// Spot config — only populate block when a meaningful spot mode is set
	if mutation.SpotType != nil && *mutation.SpotType != patching_engine.PodMutationSpotTypeSPOTTYPEUNSPECIFIED {
		spotConfig := map[string]interface{}{
			FieldPodMutationSpotMode: string(*mutation.SpotType),
		}
		if mutation.SpotDistributionPercentage != nil {
			spotConfig[FieldPodMutationSpotDistributionPct] = int(*mutation.SpotDistributionPercentage)
		}
		if err := d.Set(FieldPodMutationSpotConfig, []map[string]interface{}{spotConfig}); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.NodeTemplatesToConsolidate != nil {
		if err := d.Set(FieldPodMutationNodeTemplates, *mutation.NodeTemplatesToConsolidate); err != nil {
			return diag.FromErr(err)
		}
	}


	if mutation.Patch != nil && len(*mutation.Patch) > 0 {
		patchJSON, err := json.Marshal(*mutation.Patch)
		if err != nil {
			return diag.FromErr(fmt.Errorf("marshaling patch: %w", err))
		}
		if err := d.Set(FieldPodMutationPatch, string(patchJSON)); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.DistributionGroups != nil {
		dgs, err := flattenDistributionGroups(*mutation.DistributionGroups)
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(FieldPodMutationDistributionGroups, dgs); err != nil {
			return diag.FromErr(err)
		}
	}

	if mutation.Source != nil {
		if err := d.Set(FieldPodMutationSource, string(*mutation.Source)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func flattenObjectFilterV2(filter *patching_engine.ObjectFilterV2) []map[string]interface{} {
	m := map[string]interface{}{}

	// Workload filter
	wm := map[string]interface{}{}
	hasWorkload := false
	if filter.Names != nil {
		wm[FieldPodMutationFilterNames] = flattenMatchers(*filter.Names)
		hasWorkload = true
	}
	if filter.Namespaces != nil {
		wm[FieldPodMutationFilterNamespaces] = flattenMatchers(*filter.Namespaces)
		hasWorkload = true
	}
	if filter.Kinds != nil {
		wm[FieldPodMutationFilterKinds] = flattenMatchers(*filter.Kinds)
		hasWorkload = true
	}
	if filter.ExcludeNames != nil {
		wm[FieldPodMutationFilterExcludeNames] = flattenMatchers(*filter.ExcludeNames)
		hasWorkload = true
	}
	if filter.ExcludeNamespaces != nil {
		wm[FieldPodMutationFilterExcludeNamespaces] = flattenMatchers(*filter.ExcludeNamespaces)
		hasWorkload = true
	}
	if filter.ExcludeKinds != nil {
		wm[FieldPodMutationFilterExcludeKinds] = flattenMatchers(*filter.ExcludeKinds)
		hasWorkload = true
	}
	if hasWorkload {
		m[FieldPodMutationFilterWorkload] = []map[string]interface{}{wm}
	}

	// Pod filter
	pm := map[string]interface{}{}
	hasPod := false
	if filter.Labels != nil {
		pm[FieldPodMutationFilterLabelsFilter] = flattenLabelsFilter(filter.Labels)
		hasPod = true
	}
	if filter.ExcludeLabels != nil {
		pm[FieldPodMutationFilterExcludeLabels] = flattenLabelsFilter(filter.ExcludeLabels)
		hasPod = true
	}
	if hasPod {
		m[FieldPodMutationFilterPod] = []map[string]interface{}{pm}
	}

	return []map[string]interface{}{m}
}

func flattenMatchers(matchers []patching_engine.ObjectFilterV2Matcher) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(matchers))
	for _, m := range matchers {
		result = append(result, map[string]interface{}{
			FieldPodMutationMatcherType:  string(lo.FromPtr(m.Type)),
			FieldPodMutationMatcherValue: lo.FromPtr(m.Value),
		})
	}
	return result
}

func flattenLabelsFilter(filter *patching_engine.ObjectFilterV2LabelsFilter) []map[string]interface{} {
	m := map[string]interface{}{
		FieldPodMutationLabelsFilterOperator: string(lo.FromPtr(filter.Operator)),
	}

	if filter.Matchers != nil {
		matchersList := make([]map[string]interface{}, 0, len(*filter.Matchers))
		for _, lm := range *filter.Matchers {
			entry := map[string]interface{}{}
			if lm.Key != nil {
				entry[FieldPodMutationLabelMatcherKey] = []map[string]interface{}{
					{
						FieldPodMutationMatcherType:  string(lo.FromPtr(lm.Key.Type)),
						FieldPodMutationMatcherValue: lo.FromPtr(lm.Key.Value),
					},
				}
			}
			if lm.Value != nil {
				entry[FieldPodMutationLabelMatcherValue] = []map[string]interface{}{
					{
						FieldPodMutationMatcherType:  string(lo.FromPtr(lm.Value.Type)),
						FieldPodMutationMatcherValue: lo.FromPtr(lm.Value.Value),
					},
				}
			}
			matchersList = append(matchersList, entry)
		}
		m[FieldPodMutationLabelsFilterMatchers] = matchersList
	}

	return []map[string]interface{}{m}
}

func flattenPodMutationNodeSelector(ns *patching_engine.PatchOptions) []map[string]interface{} {
	m := map[string]interface{}{}
	if ns.Add != nil {
		m[FieldPodMutationNodeSelectorAdd] = *ns.Add
	}
	if ns.Remove != nil {
		m[FieldPodMutationNodeSelectorRemove] = *ns.Remove
	}
	return []map[string]interface{}{m}
}

func flattenTolerations(tolerations []patching_engine.Toleration) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(tolerations))
	for _, t := range tolerations {
		tolerationSeconds, err := tolerationSecondsToInt(t.TolerationSeconds)
		if err != nil {
			return nil, fmt.Errorf("converting tolerationSeconds: %w", err)
		}

		result = append(result, map[string]interface{}{
			FieldPodMutationTolerationKey:      lo.FromPtr(t.Key),
			FieldPodMutationTolerationOperator: lo.FromPtr(t.Operator),
			FieldPodMutationTolerationValue:    lo.FromPtr(t.Value),
			FieldPodMutationTolerationEffect:   lo.FromPtr(t.Effect),
			FieldPodMutationTolerationSeconds:  tolerationSeconds,
		})
	}
	return result, nil
}

func tolerationSecondsToInt(s *string) (int, error) {
	if s == nil {
		return 0, nil
	}

	v, err := strconv.Atoi(*s)
	if err != nil {
		return 0, fmt.Errorf("atoi tolerationSeconds: %w", err)
	}

	return v, nil
}

func flattenAffinity(affinity *patching_engine.Affinity) []map[string]interface{} {
	if affinity.NodeAffinity == nil {
		return nil
	}

	naMap := map[string]interface{}{}
	if affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		terms := make([]map[string]interface{}, 0)
		for _, term := range *affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			termMap := map[string]interface{}{
				FieldPodMutationWeight: int(lo.FromPtr(term.Weight)),
			}

			if term.Preference != nil && term.Preference.MatchExpressions != nil {
				exprs := make([]map[string]interface{}, 0)
				for _, req := range *term.Preference.MatchExpressions {
					exprMap := map[string]interface{}{
						FieldPodMutationMatchExpressionsKey:      lo.FromPtr(req.Key),
						FieldPodMutationMatchExpressionsOperator: lo.FromPtr(req.Operator),
					}
					if req.Values != nil {
						exprMap[FieldPodMutationValues] = *req.Values
					}
					exprs = append(exprs, exprMap)
				}
				termMap[FieldPodMutationPreference] = []map[string]interface{}{
					{FieldPodMutationMatchExpressions: exprs},
				}
			}
			terms = append(terms, termMap)
		}
		naMap[FieldPodMutationPreferred] = terms
	}

	return []map[string]interface{}{
		{
			FieldPodMutationNodeAffinity: []map[string]interface{}{naMap},
		},
	}
}

func flattenDistributionGroups(groups []patching_engine.DistributionGroup) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		gMap := map[string]interface{}{
			FieldPodMutationDistributionGroupName: lo.FromPtr(g.Name),
			FieldPodMutationDistributionGroupPct:  int(lo.FromPtr(g.Percentage)),
		}

		if g.Config != nil {
			configMap := map[string]interface{}{}
			if g.Config.Labels != nil {
				configMap[FieldPodMutationLabels] = *g.Config.Labels
			}
			if g.Config.Annotations != nil {
				configMap[FieldPodMutationAnnotations] = *g.Config.Annotations
			}
			if g.Config.NodeSelector != nil {
				configMap[FieldPodMutationNodeSelector] = flattenPodMutationNodeSelector(g.Config.NodeSelector)
			}
			if g.Config.Tolerations != nil {
				flatTolerations, err := flattenTolerations(*g.Config.Tolerations)
				if err != nil {
					return nil, fmt.Errorf("flattening tolerations: %w", err)
				}

				configMap[FieldPodMutationTolerations] = flatTolerations
			}
			if g.Config.Affinity != nil {
				configMap[FieldPodMutationAffinity] = flattenAffinity(g.Config.Affinity)
			}
			if g.Config.SpotType != nil && *g.Config.SpotType != "SPOT_TYPE_UNSPECIFIED" {
				configMap[FieldPodMutationSpotType] = string(*g.Config.SpotType)
			}
			if g.Config.NodeTemplatesToConsolidate != nil {
				configMap[FieldPodMutationNodeTemplates] = *g.Config.NodeTemplatesToConsolidate
			}
			if g.Config.Patch != nil && len(*g.Config.Patch) > 0 {
				patchJSON, err := json.Marshal(*g.Config.Patch)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal patch: %w", err)
				}

				configMap[FieldPodMutationPatch] = string(patchJSON)
			}
			gMap[FieldPodMutationDistributionGroupConfiguration] = []map[string]interface{}{configMap}
		}

		result = append(result, gMap)
	}

	return result, nil
}
