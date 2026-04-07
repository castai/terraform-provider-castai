package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	FieldPodMutationSpotConfig               = "spot_config"
	FieldPodMutationSpotMode                 = "spot_mode"
	FieldPodMutationSpotDistributionPct      = "distribution_percentage"
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

var mutationConfigSchema = map[string]*schema.Schema{
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

		Description: "CAST AI pod mutation resource allows managing pod mutations for Kubernetes workloads.",
		Schema:      s,
	}
}

func distributionGroupSchema() map[string]*schema.Schema {
	configSchema := map[string]*schema.Schema{}
	for k, v := range mutationConfigSchema {
		configSchema[k] = v
	}
	configSchema[FieldPodMutationSpotMode] = &schema.Schema{
		Type:             schema.TypeString,
		Optional:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(spotModeValues, false)),
		Description:      "Spot mode: OPTIONAL_SPOT, USE_ONLY_SPOT, or PREFERRED_SPOT.",
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
	if filterList, ok := d.Get(FieldPodMutationFilterV2).([]interface{}); ok && len(filterList) > 0 && filterList[0] != nil {
		mutation.ObjectFilterV2 = stateToObjectFilterV2(filterList[0].(map[string]interface{}))
	}

	// Patch
	if patchStr, ok := d.GetOk(FieldPodMutationPatch); ok {
		if s := patchStr.(string); s != "" {
			var patchArr []map[string]interface{}
			if err := json.Unmarshal([]byte(s), &patchArr); err == nil {
				mutation.Patch = &patchArr
			}
		}
	}

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

	// Restart policy
	// Distribution groups
	if v, ok := d.GetOk(FieldPodMutationDistributionGroups); ok {
		groups := stateToDistributionGroups(v.([]interface{}))
		mutation.DistributionGroups = &groups
	}

	return mutation
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
				config := &patching_engine.DistributionGroupConfig{}
				if v, ok := configMap[FieldPodMutationSpotMode]; ok {
					if s, ok := v.(string); ok && s != "" {
						st := patching_engine.DistributionGroupConfigSpotType(s)
						config.SpotType = &st
					}
				}
				if v, ok := configMap[FieldPodMutationPatch]; ok {
					if patchStr, ok := v.(string); ok && patchStr != "" {
						var patchArr []map[string]interface{}
						if err := json.Unmarshal([]byte(patchStr), &patchArr); err == nil {
							config.Patch = &patchArr
						}
					}
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

func flattenDistributionGroups(groups []patching_engine.DistributionGroup) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		gMap := map[string]interface{}{
			FieldPodMutationDistributionGroupName: lo.FromPtr(g.Name),
			FieldPodMutationDistributionGroupPct:  int(lo.FromPtr(g.Percentage)),
		}

		if g.Config != nil {
			configMap := map[string]interface{}{}
			if g.Config.SpotType != nil && *g.Config.SpotType != patching_engine.DistributionGroupConfigSpotTypeSPOTTYPEUNSPECIFIED {
				configMap[FieldPodMutationSpotMode] = string(*g.Config.SpotType)
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
