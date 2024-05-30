package types

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
}
