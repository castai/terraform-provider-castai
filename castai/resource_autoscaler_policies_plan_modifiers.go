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
type fillNullsFromState struct{}

func (fillNullsFromState) Description(_ context.Context) string {
	return "Fills null fields inside the section from the state."
}

func (fillNullsFromState) MarkdownDescription(ctx context.Context) string {
	return fillNullsFromState{}.Description(ctx)
}

func (fillNullsFromState) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	filled := fillNullsFromStateValue(ctx, req.PlanValue, req.StateValue)
	if !filled.Equal(req.PlanValue) {
		resp.PlanValue = filled.(types.List)
	}
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
