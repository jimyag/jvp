package virtcustomize

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		setup         func(*testing.T) *Client
		expectError   bool
		errorContains string
		validate      func(*testing.T, *Client)
	}{
		{
			name: "create client successfully",
			setup: func(t *testing.T) *Client {
				client, err := NewClient()
				if err != nil {
					// virt-customize 可能不存在，使用 mock path
					return NewClientWithPath("/usr/bin/virt-customize")
				}
				return client
			},
			expectError: false,
			validate: func(t *testing.T, c *Client) {
				assert.NotNil(t, c)
				assert.NotEmpty(t, c.virtCustomizePath)
			},
		},
		{
			name: "create client with custom path",
			setup: func(t *testing.T) *Client {
				return NewClientWithPath("/custom/path/virt-customize")
			},
			expectError: false,
			validate: func(t *testing.T, c *Client) {
				assert.NotNil(t, c)
				assert.Equal(t, "/custom/path/virt-customize", c.virtCustomizePath)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := tc.setup(t)
			if tc.validate != nil {
				tc.validate(t, client)
			}
		})
	}
}

func TestClient_ValidateDiskPath(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		diskPath      string
		setup         func(*testing.T) string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid qcow2 disk path",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				diskPath := filepath.Join(tmpDir, "test.qcow2")
				file, err := os.Create(diskPath)
				require.NoError(t, err)
				file.Close()
				return diskPath
			},
			expectError: false,
		},
		{
			name: "non-existent disk path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/disk.qcow2"
			},
			expectError:   true,
			errorContains: "disk file not found",
		},
		{
			name: "unsupported disk format",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				diskPath := filepath.Join(tmpDir, "test.raw")
				file, err := os.Create(diskPath)
				require.NoError(t, err)
				file.Close()
				return diskPath
			},
			expectError:   true,
			errorContains: "unsupported disk format",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := NewClientWithPath("/usr/bin/virt-customize")
			diskPath := tc.setup(t)

			err := client.ValidateDiskPath(diskPath)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_SetTimeout(t *testing.T) {
	t.Parallel()

	client := NewClientWithPath("/usr/bin/virt-customize")
	originalTimeout := client.timeout

	newTimeout := 10 * time.Minute
	client.SetTimeout(newTimeout)

	assert.Equal(t, newTimeout, client.timeout)
	assert.NotEqual(t, originalTimeout, client.timeout)
}
