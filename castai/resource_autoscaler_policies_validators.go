package castai

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The section fields live inside plain object types, which carry no per-field
// schema, so their constraints are enforced by section-level validators.

// clusterLimitsValidator enforces the cpu entry constraints: at most one cpu
// object, and max_cores set to at least 2.
type clusterLimitsValidator struct{}

func (clusterLimitsValidator) Description(_ context.Context) string {
	return "cpu must contain at most one entry with max_cores set to at least 2"
}

func (v clusterLimitsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (clusterLimitsValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, elem := range req.ConfigValue.Elements() {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsUnknown() {
			continue
		}
		cpuV, ok := obj.Attributes()[FieldClusterLimitsCPU]
		if !ok || cpuV.IsNull() {
			continue
		}
		cpu, ok := cpuV.(types.List)
		if !ok || cpu.IsUnknown() {
			continue
		}
		if len(cpu.Elements()) > 1 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"At most one cpu entry is allowed",
				"The cluster_limits section accepts a single cpu entry.",
			)
			continue
		}
		if len(cpu.Elements()) == 0 {
			continue
		}
		cpuAttrs, ok := cpu.Elements()[0].(types.Object)
		if !ok {
			continue
		}
		maxV, ok := cpuAttrs.Attributes()[FieldClusterLimitsCPUMaxCores]
		if !ok || maxV.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"max_cores is required",
				"The cpu entry requires max_cores.",
			)
			continue
		}
		if max, ok := maxV.(types.Int64); ok && !max.IsUnknown() && max.ValueInt64() < 2 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"max_cores must be at least 2",
				"The maximum allowed amount of vCPUs must be at least 2.",
			)
		}
	}
}

// unschedulablePodsValidator enforces at most one pod_pinner entry.
type unschedulablePodsValidator struct{}

func (unschedulablePodsValidator) Description(_ context.Context) string {
	return "pod_pinner must contain at most one entry"
}

func (v unschedulablePodsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (unschedulablePodsValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, elem := range req.ConfigValue.Elements() {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsUnknown() {
			continue
		}
		ppV, ok := obj.Attributes()[FieldUnschedulablePodsPodPinner]
		if !ok || ppV.IsNull() {
			continue
		}
		pp, ok := ppV.(types.List)
		if !ok || pp.IsUnknown() {
			continue
		}
		if len(pp.Elements()) > 1 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"At most one pod_pinner entry is allowed",
				"The unschedulable_pods section accepts a single pod_pinner entry.",
			)
		}
	}
}
