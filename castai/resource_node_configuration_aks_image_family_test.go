package castai

import (
	"testing"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/stretchr/testify/require"
)

// TestToAKSImageFamily verifies that each accepted aks_image_family string
// value maps to the correct SDK enum. A typo in the SDK constant used in any
// case branch would silently misconfigure provisioned AKS nodes.
func TestToAKSImageFamily(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		expected sdk.NodeconfigV1AKSConfigImageFamily
	}{
		"ubuntu": {
			input:    aksImageFamilyUbuntu,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU,
		},
		"ubuntu2204": {
			input:    aksImageFamilyUbuntu2204,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2204,
		},
		"ubuntu2404": {
			input:    aksImageFamilyUbuntu2404,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2404,
		},
		"ubuntu2204 uppercase normalised": {
			input:    "UBUNTU2204",
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2204,
		},
		"ubuntu2404 uppercase normalised": {
			input:    "UBUNTU2404",
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2404,
		},
		"azure-linux": {
			input:    aksImageFamilyAzureLinux,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYAZURELINUX,
		},
		"windows2019": {
			input:    aksImageFamilyWindows2019,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2019,
		},
		"windows2022": {
			input:    aksImageFamilyWindows2022,
			expected: sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2022,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			got := toAKSImageFamily(tc.input)
			r.NotNil(got, "expected non-nil family for input %q", tc.input)
			r.Equal(tc.expected, *got)
		})
	}
}

func TestToAKSImageFamily_EmptyAndUnknown(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	r.Nil(toAKSImageFamily(""))
	r.Nil(toAKSImageFamily("ubuntu1804")) // unsupported family
}

// TestFromAKSImageFamily verifies that each SDK enum maps back to the
// expected aks_image_family string. Both the UPPER_CASE ("FAMILY_*") and
// lower_case ("family_*") SDK variants are accepted by the API, so both must
// flatten to the same provider string.
func TestFromAKSImageFamily(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    sdk.NodeconfigV1AKSConfigImageFamily
		expected string
	}{
		"ubuntu upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU,
			expected: aksImageFamilyUbuntu,
		},
		"ubuntu lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyUbuntu,
			expected: aksImageFamilyUbuntu,
		},
		"ubuntu2204 upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2204,
			expected: aksImageFamilyUbuntu2204,
		},
		"ubuntu2204 lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyUbuntu2204,
			expected: aksImageFamilyUbuntu2204,
		},
		"ubuntu2404 upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU2404,
			expected: aksImageFamilyUbuntu2404,
		},
		"ubuntu2404 lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyUbuntu2404,
			expected: aksImageFamilyUbuntu2404,
		},
		"azure-linux upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYAZURELINUX,
			expected: aksImageFamilyAzureLinux,
		},
		"azure-linux lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyAzureLinux,
			expected: aksImageFamilyAzureLinux,
		},
		"windows2019 upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2019,
			expected: aksImageFamilyWindows2019,
		},
		"windows2019 lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyWindows2019,
			expected: aksImageFamilyWindows2019,
		},
		"windows2022 upper": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2022,
			expected: aksImageFamilyWindows2022,
		},
		"windows2022 lower": {
			input:    sdk.NodeconfigV1AKSConfigImageFamilyFamilyWindows2022,
			expected: aksImageFamilyWindows2022,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			r.Equal(tc.expected, fromAKSImageFamily(tc.input))
		})
	}
}

func TestFromAKSImageFamily_Unknown(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	r.Equal("", fromAKSImageFamily(sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUNSPECIFIED))
}

// TestAKSImageFamily_RoundTrip verifies that the to/from mapping is
// idempotent for every supported family string: converting a string to the
// SDK enum and back yields the original value.
func TestAKSImageFamily_RoundTrip(t *testing.T) {
	t.Parallel()

	families := []string{
		aksImageFamilyUbuntu,
		aksImageFamilyUbuntu2204,
		aksImageFamilyUbuntu2404,
		aksImageFamilyAzureLinux,
		aksImageFamilyWindows2019,
		aksImageFamilyWindows2022,
	}

	for _, family := range families {
		family := family
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			sdkFamily := toAKSImageFamily(family)
			r.NotNil(sdkFamily, "toAKSImageFamily returned nil for %q", family)

			back := fromAKSImageFamily(*sdkFamily)
			r.Equal(family, back, "round-trip mismatch for %q", family)
		})
	}
}
