package types

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// ResourceProvider defines an common interface for *schema.ResourceData and *schema.ResourceDiff
type ResourceProvider interface {
	// GetOk functions the same way as ResourceData.GetOk, but it also checks the
	// new diff levels to provide data consistent with the current state of the
	// customized diff.
	//
	// for more info: *schema.ResourceData or *schema.ResourceDiff
	GetOk(key string) (interface{}, bool)

	// GetOkExists functions the same way as GetOkExists within ResourceData, but
	// it also checks the new diff levels to provide data consistent with the
	// current state of the customized diff.
	//
	// for more info: *schema.ResourceData or *schema.ResourceDiff
	GetOkExists(key string) (interface{}, bool)

	// GetRawConfigAt returns a value from the raw config at the given path. In some cases this is necessary
	// to determine whether an optional field has been defined when accounting for potential state drift due to
	// external changes.
	GetRawConfigAt(valPath cty.Path) (cty.Value, diag.Diagnostics)
}
