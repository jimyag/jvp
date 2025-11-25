package service

import (
	"context"
	"testing"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNodeService(t *testing.T) {
	t.Parallel()

	t.Run("create node service with libvirt client", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)
		assert.NotNil(t, nodeService)
	})
}

func TestNodeService_ListNodes(t *testing.T) {
	t.Parallel()

	t.Run("list nodes returns local node", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)

		// Set up mock expectations
		ts.MockLibvirt.On("GetHostname").Return("test-host", nil)
		ts.MockLibvirt.On("GetLibvirtVersion").Return("8.0.0", nil)

		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		nodes, err := nodeService.ListNodes(ctx)

		require.NoError(t, err)
		assert.NotNil(t, nodes)
		assert.Len(t, nodes, 1)
		assert.Equal(t, "test-host", nodes[0].Name)
		assert.Equal(t, entity.NodeTypeLocal, nodes[0].Type)
		assert.Equal(t, entity.NodeStateOnline, nodes[0].State)

		ts.MockLibvirt.AssertExpectations(t)
	})
}

func TestNodeService_DescribeNode(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		nodeName    string
		expectError bool
		expectName  string
	}{
		{
			name:        "describe existing local node",
			nodeName:    "test-host",
			expectError: false,
			expectName:  "test-host",
		},
		{
			name:        "describe non-existing node",
			nodeName:    "non-existing",
			expectError: true,
			expectName:  "",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := setupTestServices(t)

			// Set up mock expectations
			ts.MockLibvirt.On("GetHostname").Return("test-host", nil)
			ts.MockLibvirt.On("GetLibvirtVersion").Return("8.0.0", nil)

			nodeService, err := NewNodeService(ts.MockLibvirt)
			require.NoError(t, err)

			ctx := context.Background()
			node, err := nodeService.DescribeNode(ctx, tc.nodeName)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, tc.expectName, node.Name)
			}

			ts.MockLibvirt.AssertExpectations(t)
		})
	}
}

func TestNodeService_DescribeNodeSummary(t *testing.T) {
	t.Parallel()

	t.Run("get node summary with valid data", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		summary, err := nodeService.DescribeNodeSummary(ctx, "test-host")

		require.NoError(t, err)
		assert.NotNil(t, summary)

		// Verify CPU info structure
		assert.Greater(t, summary.CPU.Cores, 0)
		assert.Greater(t, summary.CPU.Threads, 0)
		assert.NotEmpty(t, summary.CPU.Model)
		assert.NotEmpty(t, summary.CPU.Vendor)
		assert.Greater(t, summary.CPU.Frequency, 0)

		// Verify Memory info structure
		assert.Greater(t, summary.Memory.Total, int64(0))
		assert.GreaterOrEqual(t, summary.Memory.Available, int64(0))
		assert.GreaterOrEqual(t, summary.Memory.Used, int64(0))

		// Verify NUMA info structure
		assert.GreaterOrEqual(t, summary.NUMA.NodeCount, 0)
		assert.NotNil(t, summary.NUMA.Nodes)

		// Verify HugePages info structure
		assert.NotNil(t, summary.HugePages.PageSizes)

		// Verify Virtualization info structure
		assert.NotNil(t, summary.Virtualization)
	})
}

func TestNodeService_DescribeNodePCI(t *testing.T) {
	t.Parallel()

	t.Run("get PCI devices returns empty list", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		devices, err := nodeService.DescribeNodePCI(ctx, "test-host")

		require.NoError(t, err)
		assert.NotNil(t, devices)
		assert.Len(t, devices, 0) // Current implementation returns empty list
	})
}

func TestNodeService_DescribeNodeUSB(t *testing.T) {
	t.Parallel()

	t.Run("get USB devices returns empty list", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		devices, err := nodeService.DescribeNodeUSB(ctx, "test-host")

		require.NoError(t, err)
		assert.NotNil(t, devices)
		assert.Len(t, devices, 0) // Current implementation returns empty list
	})
}

func TestNodeService_DescribeNodeNet(t *testing.T) {
	t.Parallel()

	t.Run("get network info returns valid structure", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		netInfo, err := nodeService.DescribeNodeNet(ctx, "test-host")

		require.NoError(t, err)
		assert.NotNil(t, netInfo)
		assert.NotNil(t, netInfo.Interfaces)
		assert.NotNil(t, netInfo.Bridges)
		assert.NotNil(t, netInfo.Bonds)
		assert.NotNil(t, netInfo.SRIOV)
	})
}

func TestNodeService_DescribeNodeDisks(t *testing.T) {
	t.Parallel()

	t.Run("get disks returns empty list", func(t *testing.T) {
		t.Parallel()

		ts := setupTestServices(t)
		nodeService, err := NewNodeService(ts.MockLibvirt)
		require.NoError(t, err)

		ctx := context.Background()
		disks, err := nodeService.DescribeNodeDisks(ctx, "test-host")

		require.NoError(t, err)
		assert.NotNil(t, disks)
		assert.Len(t, disks, 0) // Current implementation returns empty list
	})
}

func TestNodeService_EnableNode(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		nodeName string
	}{
		{
			name:     "enable local node",
			nodeName: "local",
		},
		{
			name:     "enable remote node",
			nodeName: "remote1",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := setupTestServices(t)
			nodeService, err := NewNodeService(ts.MockLibvirt)
			require.NoError(t, err)

			ctx := context.Background()
			err = nodeService.EnableNode(ctx, tc.nodeName)

			assert.NoError(t, err) // Current implementation always returns nil
		})
	}
}

func TestNodeService_DisableNode(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		nodeName string
	}{
		{
			name:     "disable local node",
			nodeName: "local",
		},
		{
			name:     "disable remote node",
			nodeName: "remote1",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := setupTestServices(t)
			nodeService, err := NewNodeService(ts.MockLibvirt)
			require.NoError(t, err)

			ctx := context.Background()
			err = nodeService.DisableNode(ctx, tc.nodeName)

			assert.NoError(t, err) // Current implementation always returns nil
		})
	}
}
