package castai

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fillNullsFromState fills null fields inside computed section lists from the
// state: the API always returns a fully materialized object, so a field the
// configuration omits must adopt the stored value instead of being planned
// for removal. Filling null values never disagrees with the configuration,
// and declared (non-null) values are never overwritten.
//
// On a first create there is no prior state, but the server still fills
// omitted inner fields with defaults. The `defaults` map lets the caller
// declare those server-side defaults up front so the plan matches the new
// state and the apply does not trip "inconsistent result after apply"
// (CSU-6199).
type fillNullsFromState struct {
	// defaults supplies inner-field values for the synthesized state used on
	// first create / import when no prior state exists. Keys are inner
	// attribute names (e.g. "min_cores", "pod_pinner"); values are the
	// corresponding attr.Value the API returns when the field is omitted.
	defaults map[string]attr.Value
}

func (fillNullsFromState) Description(_ context.Context) string {
	return "Fills null fields inside the section from the state, or from configured defaults on first create."
}

func (fillNullsFromState) MarkdownDescription(ctx context.Context) string {
	return fillNullsFromState{}.Description(ctx)
}

func (m fillNullsFromState) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	stateValue := req.StateValue
	if stateValue.IsNull() || stateValue.IsUnknown() {
		synthetic, ok := syntheticStateFromDefaults(ctx, req.PlanValue, m.defaults)
		if !ok {
			return
		}
		stateValue = synthetic
	}

	filled := fillNullsFromStateValue(ctx, req.PlanValue, stateValue)
	if !filled.Equal(req.PlanValue) {
		resp.PlanValue = filled.(types.List)
	}
}

// syntheticStateFromDefaults builds a one-element state list for the planned
// section by replacing null inner fields with the configured defaults. It
// only fills nulls; declared values pass through untouched. The result is
// used in place of the prior state when none exists (first create / import).
func syntheticStateFromDefaults(ctx context.Context, planned types.List, defaults map[string]attr.Value) (types.List, bool) {
	if len(defaults) == 0 || len(planned.Elements()) == 0 {
		return types.List{}, false
	}

	plannedObj, ok := planned.Elements()[0].(types.Object)
	if !ok {
		return types.List{}, false
	}

	attrs := make(map[string]attr.Value, len(plannedObj.Attributes()))
	for k, v := range plannedObj.Attributes() {
		attrs[k] = v
	}

	for k, dv := range defaults {
		cur, ok := attrs[k]
		if !ok || cur == nil || !cur.IsNull() {
			continue
		}
		attrs[k] = dv
	}

	synthetic, diags := types.ObjectValue(plannedObj.AttributeTypes(ctx), attrs)
	if diags.HasError() {
		return types.List{}, false
	}

	out, diags := types.ListValue(planned.ElementType(ctx), []attr.Value{synthetic})
	if diags.HasError() {
		return types.List{}, false
	}
	return out, true
}

// fillNullsFromStateValue fills null plan values from the state, recursing
// through nested objects and lists.
func fillNullsFromStateValue(ctx context.Context, planned, state attr.Value) attr.Value {
	switch p := planned.(type) {
	case types.Object:
		s, ok := state.(types.Object)
		if !ok || s.IsUnknown() {
			return planned
		}
		attrs := make(map[string]attr.Value, len(p.Attributes()))
		for k, v := range p.Attributes() {
			if sv, ok := s.Attributes()[k]; ok {
				attrs[k] = fillNullsFromStateValue(ctx, v, sv)
			} else {
				attrs[k] = v
			}
		}
		if obj, diags := types.ObjectValue(p.AttributeTypes(ctx), attrs); !diags.HasError() {
			return obj
		}
		return planned
	case types.List:
		s, ok := state.(types.List)
		if !ok || s.IsUnknown() {
			return planned
		}
		if p.IsNull() {
			// A nested list the configuration omits adopts the stored value.
			return s
		}
		if len(p.Elements()) != len(s.Elements()) {
			return planned
		}
		elems := make([]attr.Value, len(p.Elements()))
		for i, e := range p.Elements() {
			elems[i] = fillNullsFromStateValue(ctx, e, s.Elements()[i])
		}
		if lst, diags := types.ListValue(p.ElementType(ctx), elems); !diags.HasError() {
			return lst
		}
		return planned
	default:
		if planned.IsNull() && state != nil && !state.IsNull() && !state.IsUnknown() {
			return state
		}
		return planned
	}
}
