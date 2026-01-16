package wallet

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"challenge/kit/db"
	"github.com/stretchr/testify/require"
)

func TestWalletFileRepository(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name string
		act  func(t *testing.T, path string)
	}{
		{
			name: "creates empty file on first load and persists set",
			act: func(t *testing.T, path string) {
				r, err := NewFileRepository(path)
				require.NoError(t, err)
				require.NoError(t, r.SetBalance(ctx, "u1", 10))

				r2, err := NewFileRepository(path)
				require.NoError(t, err)
				bal, err := r2.GetBalance(ctx, "u1")
				require.NoError(t, err)
				require.Equal(t, int64(10), bal)
			},
		},
		{
			name: "debit persists",
			act: func(t *testing.T, path string) {
				r, err := NewFileRepository(path)
				require.NoError(t, err)
				require.NoError(t, r.SetBalance(ctx, "u1", 10))
				require.NoError(t, r.DebitIfSufficientFunds(ctx, "u1", 3))

				r2, err := NewFileRepository(path)
				require.NoError(t, err)
				bal, err := r2.GetBalance(ctx, "u1")
				require.NoError(t, err)
				require.Equal(t, int64(7), bal)
			},
		},
		{
			name: "invalid json returns internal error",
			act: func(t *testing.T, path string) {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte("not-json"), 0o644))
				_, err := NewFileRepository(path)
				require.Error(t, err)
				require.ErrorIs(t, err, db.ErrInternal)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "wallet.json")
			tt.act(t, path)
		})
	}
}
