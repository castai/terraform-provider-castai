package commitments

import (
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

// CastaiGCPCommitmentImport is a wrapper around sdk.CastaiInventoryV1beta1GCPCommitmentImport implementing the cud interface
type CastaiGCPCommitmentImport struct {
	sdk.CastaiInventoryV1beta1GCPCommitmentImport
}

var _ cud = CastaiGCPCommitmentImport{}

func (c CastaiGCPCommitmentImport) getCUDKey() cudConfigMatcherKey {
	return cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
		typ:    lo.FromPtr(c.Type),
	}
}

// CastaiCommitment is a wrapper around sdk.CastaiInventoryV1beta1Commitment implementing the cud interface
type CastaiCommitment struct {
	sdk.CastaiInventoryV1beta1Commitment
}

var _ cud = CastaiCommitment{}

func (c CastaiCommitment) getCUDKey() cudConfigMatcherKey {
	res := cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
	}
	if c.GcpResourceCudContext != nil {
		res.typ = *c.GcpResourceCudContext.Type
	}
	return res
}
