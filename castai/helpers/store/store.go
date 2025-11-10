package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PrivateState interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

func NewWriteOnlyStore(s PrivateState, key string) *WriteOnlyStore {
	return &WriteOnlyStore{
		state: s,
		key:   key,
	}
}

type WriteOnlyStore struct {
	state PrivateState
	key   string
}

func (w *WriteOnlyStore) Equal(ctx context.Context, configValue types.String) (bool, diag.Diagnostics) {
	val, diags := w.state.GetKey(ctx, w.key)
	var v string
	if err := json.Unmarshal(val, &v); err != nil {
		diags.AddError(fmt.Sprintf("failed to unmarshal value for `%s`", w.key), err.Error())
	}

	hashedValue := generateSHA256Hash(configValue.ValueString())
	return v == hashedValue, diags
}

func (w *WriteOnlyStore) Set(ctx context.Context, value types.String) diag.Diagnostics {
	if value.IsNull() {
		return w.state.SetKey(ctx, w.key, []byte(""))
	}

	hashedValue := generateSHA256Hash(value.ValueString())
	return w.state.SetKey(ctx, w.key, fmt.Appendf(nil, `"%s"`, hashedValue))
}

func generateSHA256Hash(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
