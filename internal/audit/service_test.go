package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"challenge/kit/observability"
	"github.com/stretchr/testify/require"
)

func TestService_Close(t *testing.T) {
	var tests = []struct {
		name string
		svc  func(t *testing.T) *Service
	}{
		{
			name: "close nil file",
			svc: func(t *testing.T) *Service {
				return NewService(observability.NewLogger())
			},
		},
		{
			name: "close with file",
			svc: func(t *testing.T) *Service {
				dir := t.TempDir()
				path := filepath.Join(dir, "audit.jsonl")
				svc, err := NewServiceWithFile(observability.NewLogger(), path)
				require.NoError(t, err)
				return svc
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.svc(t)
			require.NotPanics(t, func() { _ = svc.Close() })
		})
	}
}

func TestService_Record(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name   string
		svc    func(t *testing.T) *Service
		assert func(t *testing.T, dir string)
	}{
		{
			name: "nil logger does nothing",
			svc: func(t *testing.T) *Service {
				return NewService(nil)
			},
			assert: func(t *testing.T, dir string) {
				_ = dir
			},
		},
		{
			name: "writes to file when configured",
			svc: func(t *testing.T) *Service {
				dir := t.TempDir()
				path := filepath.Join(dir, "audit.jsonl")
				svc, err := NewServiceWithFile(logger, path)
				require.NoError(t, err)
				return svc
			},
			assert: func(t *testing.T, dir string) {
				_ = dir
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.svc(t)
			require.NotPanics(t, func() {
				svc.Record(ctx, "event", map[string]any{"k": "v"})
			})

			// if service has a file, verify it is non-empty
			svc.fileMu.Lock()
			f := svc.f
			svc.fileMu.Unlock()
			if f != nil {
				info, err := os.Stat(f.Name())
				require.NoError(t, err)
				require.Greater(t, info.Size(), int64(0))
			}
		})
	}
}
