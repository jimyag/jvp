package service

import (
	"context"
	"fmt"
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

			store := newTemplateStoreForTest(t)
			mockLibvirt := libvirt.NewMockClient()

			mockLibvirt.On("GetVolume", "images", "base-template.qcow2").
				Return(&libvirt.VolumeInfo{
					Name:        "base-template.qcow2",
					Path:        "/var/lib/libvirt/images/base-template.qcow2",
					CapacityB:   40 * 1024 * 1024 * 1024,
					AllocationB: 20 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).
				Once()

			nodeProvider := &sequencedNodeProvider{
				t:             t,
				client:        mockLibvirt,
				expectedNodes: []string{tc.expectNode},
			}

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

			template, err := service.RegisterTemplate(ctx, req)
			require.NoError(t, err)
			assert.Equal(t, tc.expectNode, template.NodeName)
			assert.Equal(t, req.Name, template.Name)
			assert.Equal(t, uint64(40), template.SizeGB)

			listResp, err := service.ListTemplates(ctx, &entity.ListTemplatesRequest{
				NodeName: tc.listNodeName,
			})
			require.NoError(t, err)
			require.Len(t, listResp, 1)
			assert.Equal(t, template.ID, listResp[0].ID)

			filterResp, err := service.ListTemplates(ctx, &entity.ListTemplatesRequest{
				PoolName: "other",
			})
			require.NoError(t, err)
			assert.Len(t, filterResp, 0)

			desc, err := service.DescribeTemplate(ctx, &entity.DescribeTemplateRequest{
				NodeName:   tc.expectNode,
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

			store := newTemplateStoreForTest(t)
			mockLibvirt := libvirt.NewMockClient()
			mockLibvirt.On("GetVolume", "images", "tmpl.qcow2").
				Return(&libvirt.VolumeInfo{
					Name:        "tmpl.qcow2",
					Path:        "/var/lib/libvirt/images/tmpl.qcow2",
					CapacityB:   10 * 1024 * 1024 * 1024,
					AllocationB: 5 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).
				Once()

			if tc.deleteVolume {
				mockLibvirt.On("DeleteVolume", "images", "tmpl.qcow2").Return(nil).Once()
			}

			expectedNodes := []string{"local"}
			if tc.deleteVolume {
				expectedNodes = append(expectedNodes, "local")
			}
			nodeProvider := &sequencedNodeProvider{
				t:             t,
				client:        mockLibvirt,
				expectedNodes: expectedNodes,
			}

			service := NewTemplateService(nodeProvider.Get, store)
			ctx := context.Background()

			template, err := service.RegisterTemplate(ctx, &entity.RegisterTemplateRequest{
				NodeName:   "",
				PoolName:   "images",
				VolumeName: "tmpl.qcow2",
				Name:       "debian-base",
			})
			require.NoError(t, err)

			newDesc := "Updated desc"
			newTags := []string{"debian"}
			newFeatures := entity.TemplateFeatures{
				CloudInit:      true,
				QemuGuestAgent: true,
			}
			newOS := entity.TemplateOS{Name: "debian", Version: "12", Arch: "x86_64"}
			updated, err := service.UpdateTemplate(ctx, &entity.UpdateTemplateRequest{
				NodeName:    "local",
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
				TemplateID:   template.ID,
				DeleteVolume: tc.deleteVolume,
			})
			require.NoError(t, err)

			listResp, err := service.ListTemplates(ctx, &entity.ListTemplatesRequest{
				NodeName: "local",
			})
			require.NoError(t, err)
			assert.Len(t, listResp, 0)

			mockLibvirt.AssertExpectations(t)
		})
	}
}

func newTemplateStoreForTest(t *testing.T) *TemplateStore {
	t.Helper()
	store, err := NewTemplateStore(t.TempDir())
	require.NoError(t, err)
	return store
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
