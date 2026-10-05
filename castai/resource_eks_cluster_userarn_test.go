package castai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEKSUserARNResourceImporter(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	res := resourceEKSClusterUserARN()
	d := res.Data(nil)
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	d.SetId(clusterID)

	result, err := res.Importer.StateContext(t.Context(), d, nil)

	r.NoError(err)
	r.Len(result, 1)
	r.Equal(clusterID, d.Id())
	r.Equal(clusterID, d.Get(EKSClusterUserARNFieldClusterID))
}
