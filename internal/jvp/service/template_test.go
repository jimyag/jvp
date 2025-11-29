package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateService_RegisterListDescribe(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name         string
		nodeName     string
		expectNode   string
		listNodeName string
	}

	testcases := []testCase{
		{
			name:         "local node default",
			nodeName:     "",
			expectNode:   "local",
			listNodeName: "",
		},
		{
			name:         "remote node explicit",
			nodeName:     "edge-node",
			expectNode:   "edge-node",
			listNodeName: "edge-node",
		},
	}

	for _, tc := range testcases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockLibvirt := libvirt.NewMockClient()
			tempDir := t.TempDir()

			// 创建 _templates_ 目录和测试文件
			templatesDir := filepath.Join(tempDir, TemplatesDirName)
			err := os.MkdirAll(templatesDir, 0o755)
			require.NoError(t, err)

			// 创建一个模拟的模板文件（实际文件，因为 lookupVolume 会直接检查文件）
			templateFile := filepath.Join(templatesDir, "base-template.qcow2")
			testData := make([]byte, 1024) // 1KB 测试文件
			err = os.WriteFile(templateFile, testData, 0o644)
			require.NoError(t, err)

			// GetStoragePool is called multiple times by TemplateStore and lookupVolume
			mockLibvirt.On("GetStoragePool", "images").
				Return(&libvirt.StoragePoolInfo{
					Name:  "images",
					State: "Active",
					Path:  tempDir,
				}, nil).
				Maybe()

			// IsRemoteConnection is called by TemplateStore and lookupVolume
			mockLibvirt.On("IsRemoteConnection").Return(false).Maybe()

			nodeProvider := &sequencedNodeProvider{
				t:             t,
				client:        mockLibvirt,
				expectedNodes: make([]string, 20), // enough for multiple calls
			}
			// Fill with expected node name
			for i := range nodeProvider.expectedNodes {
				nodeProvider.expectedNodes[i] = tc.expectNode
			}

			store := newTemplateStoreForTest(t, nodeProvider.Get)
			service := NewTemplateService(nodeProvider.Get, store)

			ctx := context.Background()
			req := &entity.RegisterTemplateRequest{
				NodeName:    tc.nodeName,
				PoolName:    "images",
				VolumeName:  "base-template.qcow2",
				Name:        fmt.Sprintf("ubuntu-%s", tc.name),
				Description: "Ubuntu template",
				Tags:        []string{"ubuntu", "cloud"},
				OS: entity.TemplateOS{
					Name:    "ubuntu",
					Version: "22.04",
					Arch:    "x86_64",
				},
				Features: entity.TemplateFeatures{
					CloudInit: true,
					Virtio:    true,
				},
			}

			result, err := service.RegisterTemplate(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, result.Template)
			template := result.Template
			assert.Equal(t, tc.expectNode, template.NodeName)
			assert.Equal(t, req.Name, template.Name)
			// SizeBytes 应该是实际文件大小（1024 bytes），SizeGB 为很小的值（因为不足 1GB）
			assert.Equal(t, uint64(1024), template.SizeBytes)
			assert.InDelta(t, float64(1024)/(1024*1024*1024), template.SizeGB, 0.0001)

			listResp, err := service.ListTemplates(ctx, &entity.ListTemplatesRequest{
				NodeName: tc.listNodeName,
				PoolName: "images",
			})
			require.NoError(t, err)
			require.Len(t, listResp, 1)
			assert.Equal(t, template.ID, listResp[0].ID)

			desc, err := service.DescribeTemplate(ctx, &entity.DescribeTemplateRequest{
				NodeName:   tc.expectNode,
				PoolName:   "images",
				TemplateID: template.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, template.ID, desc.ID)

			mockLibvirt.AssertExpectations(t)
		})
	}
}

func TestTemplateService_UpdateAndDelete(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name         string
		deleteVolume bool
	}

	testcases := []testCase{
		{name: "delete metadata only", deleteVolume: false},
		{name: "delete metadata and volume", deleteVolume: true},
	}

	for _, tc := range testcases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockLibvirt := libvirt.NewMockClient()
			tempDir := t.TempDir()

			// 创建 _templates_ 目录和测试文件
			templatesDir := filepath.Join(tempDir, TemplatesDirName)
			err := os.MkdirAll(templatesDir, 0o755)
			require.NoError(t, err)

			// 创建一个模拟的模板文件
			templateFile := filepath.Join(templatesDir, "tmpl.qcow2")
			testData := make([]byte, 1024) // 1KB 测试文件
			err = os.WriteFile(templateFile, testData, 0o644)
			require.NoError(t, err)

			if tc.deleteVolume {
				mockLibvirt.On("DeleteVolume", "images", "tmpl.qcow2").Return(nil).Once()
			}

			// GetStoragePool is called multiple times by TemplateStore and lookupVolume
			mockLibvirt.On("GetStoragePool", "images").
				Return(&libvirt.StoragePoolInfo{
					Name:  "images",
					State: "Active",
					Path:  tempDir,
				}, nil).
				Maybe()

			// IsRemoteConnection is called by TemplateStore and lookupVolume
			mockLibvirt.On("IsRemoteConnection").Return(false).Maybe()

			nodeProvider := &sequencedNodeProvider{
				t:             t,
				client:        mockLibvirt,
				expectedNodes: make([]string, 20), // enough for multiple calls
			}
			// Fill with "local" since all calls expect "local"
			for i := range nodeProvider.expectedNodes {
				nodeProvider.expectedNodes[i] = "local"
			}

			store := newTemplateStoreForTest(t, nodeProvider.Get)
			service := NewTemplateService(nodeProvider.Get, store)
			ctx := context.Background()

			result, err := service.RegisterTemplate(ctx, &entity.RegisterTemplateRequest{
				NodeName:   "",
				PoolName:   "images",
				VolumeName: "tmpl.qcow2",
				Name:       "debian-base",
			})
			require.NoError(t, err)
			require.NotNil(t, result.Template)
			template := result.Template

			newDesc := "Updated desc"
			newTags := []string{"debian"}
			newFeatures := entity.TemplateFeatures{
				CloudInit:      true,
				QemuGuestAgent: true,
			}
			newOS := entity.TemplateOS{Name: "debian", Version: "12", Arch: "x86_64"}
			updated, err := service.UpdateTemplate(ctx, &entity.UpdateTemplateRequest{
				NodeName:    "local",
				PoolName:    "images",
				TemplateID:  template.ID,
				Description: &newDesc,
				Tags:        &newTags,
				Features:    &newFeatures,
				OS:          &newOS,
			})
			require.NoError(t, err)
			assert.Equal(t, newDesc, updated.Description)
			assert.Equal(t, newTags, updated.Tags)
			assert.Equal(t, newFeatures, updated.Features)
			assert.Equal(t, newOS, updated.OS)

			err = service.DeleteTemplate(ctx, &entity.DeleteTemplateRequest{
				NodeName:     "local",
				PoolName:     "images",
				TemplateID:   template.ID,
				DeleteVolume: tc.deleteVolume,
			})
			require.NoError(t, err)

			listResp, err := service.ListTemplates(ctx, &entity.ListTemplatesRequest{
				NodeName: "local",
				PoolName: "images",
			})
			require.NoError(t, err)
			assert.Len(t, listResp, 0)

			mockLibvirt.AssertExpectations(t)
		})
	}
}

func newTemplateStoreForTest(t *testing.T, nodeProvider NodeStorageGetter) *TemplateStore {
	t.Helper()
	return NewTemplateStore(nodeProvider)
}

type sequencedNodeProvider struct {
	t             *testing.T
	client        libvirt.LibvirtClient
	expectedNodes []string
}

func (p *sequencedNodeProvider) Get(ctx context.Context, nodeName string) (libvirt.LibvirtClient, error) {
	require.NotEmpty(p.t, p.expectedNodes, "unexpected GetNodeStorage call")
	expected := p.expectedNodes[0]
	p.expectedNodes = p.expectedNodes[1:]
	assert.Equal(p.t, expected, nodeName)
	return p.client, nil
}
