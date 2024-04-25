package commitments

import (
	"path"

	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

// CastaiGCPCommitmentImport is a wrapper around sdk.CastaiInventoryV1beta1GCPCommitmentImport implementing the cud interface
type CastaiGCPCommitmentImport struct {
	sdk.CastaiInventoryV1beta1GCPCommitmentImport
}

var _ commitment = CastaiGCPCommitmentImport{}

func (c CastaiGCPCommitmentImport) getKey() commitmentConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	return commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
		typ:    lo.FromPtr(c.Type),
	}
}

// CastaiAzureReservationImport is a wrapper around sdk.CastaiInventoryV1beta1AzureReservationImport implementing the cud interface
type CastaiAzureReservationImport struct {
	sdk.CastaiInventoryV1beta1AzureReservationImport
}

var _ commitment = CastaiAzureReservationImport{}

func (c CastaiAzureReservationImport) getKey() commitmentConfigMatcherKey {
	return commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: lo.FromPtr(c.Region),
		typ:    lo.FromPtr(c.ProductName),
	}
}

// CastaiCommitment is a wrapper around sdk.CastaiInventoryV1beta1Commitment implementing the cud interface
type CastaiCommitment struct {
	sdk.CastaiInventoryV1beta1Commitment
}

var _ commitment = CastaiCommitment{}

func (c CastaiCommitment) getKey() commitmentConfigMatcherKey {
	var region string
	if c.Region != nil {
		_, region = path.Split(*c.Region)
	}
	res := commitmentConfigMatcherKey{
		name:   lo.FromPtr(c.Name),
		region: region,
	}
	if c.GcpResourceCudContext != nil {
		res.typ = *c.GcpResourceCudContext.Type
	}
	if c.AzureReservationContext != nil {
		res.typ = *c.AzureReservationContext.InstanceType
	}
	return res
}
