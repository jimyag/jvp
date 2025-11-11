package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Run(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		ctx         context.Context
		expectError bool
	}{
		{
			name:        "run successfully",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "run with timeout context",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
				return ctx
			}(),
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, err := New()
			require.NoError(t, err)

			// Run 方法在当前实现中只是返回 nil，所以这里测试它不会出错
			err = service.Run(tc.ctx)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_Shutdown(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		ctx         context.Context
		expectError bool
	}{
		{
			name:        "shutdown successfully",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "shutdown with timeout context",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
				return ctx
			}(),
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, err := New()
			require.NoError(t, err)

			// Shutdown 方法在当前实现中只是返回 nil，所以这里测试它不会出错
			err = service.Shutdown(tc.ctx)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_New(t *testing.T) {
	t.Parallel()

	service, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, service)
}
