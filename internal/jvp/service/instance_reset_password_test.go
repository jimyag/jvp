package service

import (
	"context"
	"strings"
	"testing"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type stubNodeProvider struct {
	client libvirt.LibvirtClient
}

func (s *stubNodeProvider) GetNodeStorage(_ context.Context, _ string) (libvirt.LibvirtClient, error) {
	return s.client, nil
}

func TestInstanceService_ResetPassword_RemoteNodeRunsVirtCustomizeOverSSH(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
	}{
		{name: "remote node executes virt-customize via ssh and skips local client"},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockLibvirt := new(libvirt.MockClient)
			mockVirt := new(virtcustomize.MockClient)

			domain := libvirtlib.Domain{Name: "i-123"}

			// GetInstance uses these calls
			mockLibvirt.On("GetDomainByName", "i-123").Return(domain, nil).Once()
			mockLibvirt.On("GetDomainInfo", domain.UUID).Return(&libvirt.DomainInfo{
				Name:      "i-123",
				UUID:      "uuid-123",
				State:     "stopped",
				Memory:    2048 * 1024,
				VCPUs:     2,
				Autostart: false,
			}, nil).Once()
			mockLibvirt.On("GetDomainState", domain).Return(uint8(5), uint32(0), nil).Once()

			mockLibvirt.On("IsRemoteConnection").Return(true).Once()
			mockLibvirt.On("GetDomainDisks", "i-123").Return([]libvirt.DomainDisk{
				{Source: libvirt.DomainDiskSource{File: "/var/lib/remote/disk.qcow2"}},
			}, nil).Once()
			mockLibvirt.On("ExecuteRemoteCommand", mock.MatchedBy(func(cmd string) bool {
				return cmd == "test -f '/var/lib/remote/disk.qcow2'"
			})).Return(nil).Once()
			mockLibvirt.On("ExecuteRemoteCommand", mock.MatchedBy(func(cmd string) bool {
				return strings.Contains(cmd, "virt-customize") && strings.Contains(cmd, "/var/lib/remote/disk.qcow2") && strings.Contains(cmd, "root:password:newpass")
			})).Return(nil).Once()

			service, err := NewInstanceService(&stubNodeProvider{client: mockLibvirt}, nil, nil)
			assert.NoError(t, err)
			service.virtCustomizeClient = mockVirt
			service.asyncRun = func(f func()) { f() }

			resp, err := service.ResetPassword(ctx, &entity.ResetPasswordRequest{
				NodeName:   "remote-node",
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{Username: "root", NewPassword: "newpass"},
				},
				AutoStart: false,
			})

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Success)

			// Local virt-customize client should not be used for remote paths
			mockVirt.AssertNotCalled(t, "ValidateDiskPath", mock.Anything)
			mockVirt.AssertNotCalled(t, "ResetMultiplePasswords", mock.Anything, mock.Anything, mock.Anything)

			mockLibvirt.AssertExpectations(t)
			mockVirt.AssertExpectations(t)
		})
	}
}
