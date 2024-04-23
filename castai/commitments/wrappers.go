package commitments

import (
	"github.com/samber/lo"
	"path"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

// CastaiGCPCommitmentImport is a wrapper around sdk.CastaiInventoryV1beta1GCPCommitmentImport implementing the cud interface
type CastaiGCPCommitmentImport struct {
	sdk.CastaiInventoryV1beta1GCPCommitmentImport
}

var _ cud = CastaiGCPCommitmentImport{}

func (c CastaiGCPCommitmentImport) getCUDKey() cudConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	return cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
		typ:    lo.FromPtr(c.Type),
	}
}

// CastaiCommitment is a wrapper around sdk.CastaiInventoryV1beta1Commitment implementing the cud interface
type CastaiCommitment struct {
	sdk.CastaiInventoryV1beta1Commitment
}

var _ cud = CastaiCommitment{}

func (c CastaiCommitment) getCUDKey() cudConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	res := cudConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
	}
	if c.GcpResourceCudContext != nil {
		res.typ = *c.GcpResourceCudContext.Type
	}
	return res
}
