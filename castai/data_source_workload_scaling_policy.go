package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldPolicyID   = "policy_id"
	FieldIsDefault  = "is_default"
	FieldIsReadonly = "is_readonly"
	FieldIsCastware = "is_castware"
)

func dataSourceWorkloadScalingPolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorkloadScalingPolicyRead,
		Description: "Fetches a single workload scaling policy by id or name, including system (read-only) " +
			"policies. Returns the full policy definition, which can be used as a template for defining " +
			"custom policies without keeping policy definitions as static code.",
		Schema: workloadScalingPolicyDataSourceSchema(),
	}
}

func workloadScalingPolicyDataSourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		FieldClusterID: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      "CAST AI cluster id.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
		},
		FieldPolicyID: {
			Type:             schema.TypeString,
			Optional:         true,
			ExactlyOneOf:     []string{FieldPolicyID, "name"},
			Description:      "Scaling policy id to fetch. Exactly one of policy_id or name must be set.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
		},
		"name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ExactlyOneOf: []string{FieldPolicyID, "name"},
			Description: "Scaling policy name to fetch. After the read, contains the name of the fetched policy. " +
				"Exactly one of policy_id or name must be set.",
		},
		FieldIsDefault: {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether this is the default scaling policy for the cluster.",
		},
		FieldIsReadonly: {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether this policy is read-only (cannot be updated or deleted).",
		},
		FieldIsCastware: {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether this policy is managed by CAST AI and only applies to castware workloads.",
		},
	}
	for k, v := range workloadScalingPolicyDefinitionSchema() {
		s[k] = v
	}
	return s
}

// workloadScalingPolicyDefinitionSchema returns the definition fields of the
// castai_workload_scaling_policy resource as computed-only attributes, so that
// the data source exposes the same shape the resource consumes.
func workloadScalingPolicyDefinitionSchema() map[string]*schema.Schema {
	out := map[string]*schema.Schema{}
	for k, v := range resourceWorkloadScalingPolicy().Schema {
		if k == FieldClusterID || k == "name" {
			continue // provided as data source inputs instead
		}
		out[k] = computedOnly(v)
	}
	return out
}

// computedOnly deep-copies s into a computed-only attribute, recursively
// converting nested blocks. Defaults and validators are dropped since nothing
// is configurable.
func computedOnly(s *schema.Schema) *schema.Schema {
	c := *s
	c.Required = false
	c.Optional = false
	c.Computed = true
	c.ForceNew = false
	c.Default = nil
	c.MinItems = 0
	c.MaxItems = 0
	c.ValidateDiagFunc = nil
	c.ValidateFunc = nil
	c.DiffSuppressFunc = nil
	c.ConflictsWith = nil
	c.ExactlyOneOf = nil
	c.AtLeastOneOf = nil
	c.RequiredWith = nil
	if res, ok := c.Elem.(*schema.Resource); ok {
		c.Elem = computedOnlyResource(res)
	}
	return &c
}

func computedOnlyResource(r *schema.Resource) *schema.Resource {
	out := &schema.Resource{Schema: make(map[string]*schema.Schema, len(r.Schema))}
	for k, v := range r.Schema {
		out.Schema[k] = computedOnly(v)
	}
	return out
}

func dataSourceWorkloadScalingPolicyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)

	var sp *sdk.WorkloadoptimizationV1WorkloadScalingPolicy

	if policyID, ok := d.GetOk(FieldPolicyID); ok {
		resp, err := client.WorkloadOptimizationAPIGetWorkloadScalingPolicyWithResponse(ctx, clusterID, policyID.(string))
		if e := sdk.CheckOKResponse(resp, err); e != nil {
			return diag.FromErr(fmt.Errorf("reading scaling policy: %w", e))
		}
		sp = resp.JSON200
	} else {
		name := d.Get("name").(string)
		if name == "" {
			return diag.Errorf("either policy_id or name must be set")
		}
		list, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
		if e := sdk.CheckOKResponse(list, err); e != nil {
			return diag.FromErr(fmt.Errorf("listing scaling policies: %w", e))
		}
		matchIdx := -1
		matches := 0
		for i, p := range list.JSON200.Items {
			if p.Name == name {
				matches++
				matchIdx = i
			}
		}
		switch matches {
		case 0:
			return diag.Errorf("scaling policy %q not found in cluster %s", name, clusterID)
		case 1:
			sp = &list.JSON200.Items[matchIdx]
		default:
			return diag.Errorf("found %d scaling policies named %q in cluster %s, use policy_id to select one", matches, name, clusterID)
		}
	}

	// A nil payload can occur if the API returns a 2xx response with an empty
	// body; treat it as not found rather than flattening an empty policy.
	if sp == nil || sp.Id == "" {
		return diag.Errorf("scaling policy not found")
	}

	d.SetId(sp.Id)
	if err := flattenWorkloadScalingPolicy(d, sp); err != nil {
		return diag.FromErr(err)
	}

	for k, v := range map[string]any{
		FieldIsDefault:  sp.IsDefault,
		FieldIsReadonly: sp.IsReadonly,
		FieldIsCastware: sp.IsCastware,
	} {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", k, err))
		}
	}

	return nil
}
