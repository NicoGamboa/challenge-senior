package recovery

import (
	"context"
	"testing"

	"challenge/kit/observability"
	"github.com/stretchr/testify/require"
)

func TestService_SendToDLQ(t *testing.T) {
	var tests = []struct {
		name string
		svc  func() *Service
	}{
		{
			name: "nil logger does not panic",
			svc: func() *Service {
				return NewService(nil)
			},
		},
		{
			name: "logger set does not panic",
			svc: func() *Service {
				return NewService(observability.NewLogger())
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.svc()
			require.NotPanics(t, func() {
				svc.SendToDLQ(context.Background(), "topic", "reason", map[string]any{"k": "v"})
			})
		})
	}
}
