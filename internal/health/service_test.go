package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHealthService_Check(t *testing.T) {
	// given:
	var tests = []struct {
		name        string
		service     func() *Service
		expectedOK  bool
		expected    map[string]string
		expectedErr string
		sleep       time.Duration
		expectCache bool
	}{
		{
			name: "caches within ttl",
			service: func() *Service {
				return NewService(50*time.Millisecond, map[string]CheckFunc{
					"dep": func(ctx context.Context) error { return nil },
				})
			},
			expectedOK:  true,
			expected:    map[string]string{"dep": "ok"},
			expectedErr: "",
			sleep:       60 * time.Millisecond,
			expectCache: true,
		},
		{
			name: "reports failure",
			service: func() *Service {
				return NewService(0, map[string]CheckFunc{
					"dep": func(ctx context.Context) error { return errors.New("boom") },
				})
			},
			expectedOK:  false,
			expected:    map[string]string{"dep": "boom"},
			expectedErr: "boom",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()

			res1 := svc.Check(context.Background())
			res2 := svc.Check(context.Background())

			require.Equal(t, tt.expectedOK, res1.OK)
			require.Equal(t, tt.expectedOK, res2.OK)
			require.Equal(t, tt.expected, res1.Checks)

			if tt.expectedErr != "" {
				require.EqualError(t, errors.New(res1.Checks["dep"]), tt.expectedErr)
				return
			}

			if tt.expectCache {
				// cache assertion: within ttl, At should be the same
				require.Equal(t, res1.At, res2.At)

				time.Sleep(tt.sleep)
				res3 := svc.Check(context.Background())
				require.NotEqual(t, res2.At, res3.At)
			}
		})
	}
}
