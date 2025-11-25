package libvirt

import (
	"github.com/digitalocean/go-libvirt"
	"github.com/stretchr/testify/mock"
)

// MockClient 是 LibvirtClient 的 mock 实现
// 用于测试，不需要真实的 libvirt 连接
type MockClient struct {
	mock.Mock
}

// 连接信息
func (m *MockClient) GetHostname() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockClient) GetLibvirtVersion() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockClient) GetNodeInfo() (*NodeInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*NodeInfo), args.Error(1)
}

func (m *MockClient) GetCapabilities() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockClient) GetSysinfo() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// Domain 操作
func (m *MockClient) GetVMSummaries() ([]libvirt.Domain, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]libvirt.Domain), args.Error(1)
}

func (m *MockClient) GetDomainInfo(domainUUID libvirt.UUID) (*DomainInfo, error) {
	args := m.Called(domainUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DomainInfo), args.Error(1)
}

func (m *MockClient) GetDomainByName(name string) (libvirt.Domain, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return libvirt.Domain{}, args.Error(1)
	}
	return args.Get(0).(libvirt.Domain), args.Error(1)
}

func (m *MockClient) GetDomainState(domain libvirt.Domain) (uint8, uint32, error) {
	args := m.Called(domain)
	return args.Get(0).(uint8), args.Get(1).(uint32), args.Error(2)
}

func (m *MockClient) CreateDomain(config *CreateVMConfig, autoStart bool) (libvirt.Domain, error) {
	args := m.Called(config, autoStart)
	if args.Get(0) == nil {
		return libvirt.Domain{}, args.Error(1)
	}
	return args.Get(0).(libvirt.Domain), args.Error(1)
}

func (m *MockClient) StartDomain(domain libvirt.Domain) error {
	args := m.Called(domain)
	return args.Error(0)
}

func (m *MockClient) StopDomain(domain libvirt.Domain) error {
	args := m.Called(domain)
	return args.Error(0)
}

func (m *MockClient) RebootDomain(domain libvirt.Domain) error {
	args := m.Called(domain)
	return args.Error(0)
}

func (m *MockClient) DestroyDomain(domain libvirt.Domain) error {
	args := m.Called(domain)
	return args.Error(0)
}

func (m *MockClient) DeleteDomain(domain libvirt.Domain, flags libvirt.DomainUndefineFlagsValues) error {
	args := m.Called(domain, flags)
	return args.Error(0)
}

func (m *MockClient) ModifyDomainMemory(domain libvirt.Domain, memoryKB uint64, live bool) error {
	args := m.Called(domain, memoryKB, live)
	return args.Error(0)
}

func (m *MockClient) ModifyDomainVCPU(domain libvirt.Domain, vcpus uint16, live bool) error {
	args := m.Called(domain, vcpus, live)
	return args.Error(0)
}

// Domain 磁盘操作
func (m *MockClient) AttachDiskToDomain(domainName, volumePath, device string) error {
	args := m.Called(domainName, volumePath, device)
	return args.Error(0)
}

func (m *MockClient) DetachDiskFromDomain(domainName, device string) error {
	args := m.Called(domainName, device)
	return args.Error(0)
}

func (m *MockClient) GetDomainDisks(domainName string) ([]DomainDisk, error) {
	args := m.Called(domainName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DomainDisk), args.Error(1)
}

// Storage Pool 操作
func (m *MockClient) GetStoragePool(poolName string) (*StoragePoolInfo, error) {
	args := m.Called(poolName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StoragePoolInfo), args.Error(1)
}

func (m *MockClient) ListStoragePools() ([]*StoragePoolInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*StoragePoolInfo), args.Error(1)
}

func (m *MockClient) EnsureStoragePool(poolName, poolType, poolPath string) error {
	args := m.Called(poolName, poolType, poolPath)
	return args.Error(0)
}

func (m *MockClient) CreateStoragePool(poolName, poolType, poolPath string) error {
	args := m.Called(poolName, poolType, poolPath)
	return args.Error(0)
}

// Storage Volume 操作
func (m *MockClient) GetVolume(poolName, volumeName string) (*VolumeInfo, error) {
	args := m.Called(poolName, volumeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VolumeInfo), args.Error(1)
}

func (m *MockClient) ListVolumes(poolName string) ([]*VolumeInfo, error) {
	args := m.Called(poolName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*VolumeInfo), args.Error(1)
}

func (m *MockClient) CreateVolume(poolName, volumeName string, sizeGB uint64, format string) (*VolumeInfo, error) {
	args := m.Called(poolName, volumeName, sizeGB, format)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VolumeInfo), args.Error(1)
}

func (m *MockClient) DeleteVolume(poolName, volumeName string) error {
	args := m.Called(poolName, volumeName)
	return args.Error(0)
}

func (m *MockClient) QemuAgentCommand(domain libvirt.Domain, command string, timeout uint32, flags uint32) (string, error) {
	args := m.Called(domain, command, timeout, flags)
	return args.String(0), args.Error(1)
}

func (m *MockClient) CheckGuestAgentAvailable(domain libvirt.Domain) (bool, error) {
	args := m.Called(domain)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) GetDomainConsoleInfo(domain libvirt.Domain) (*ConsoleInfo, error) {
	args := m.Called(domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConsoleInfo), args.Error(1)
}

// Snapshot 操作
func (m *MockClient) ListSnapshots(domainName string) ([]string, error) {
	args := m.Called(domainName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// NewMockClient 创建新的 MockClient
// 这是一个便捷函数，用于在测试中创建 mock client
func NewMockClient() *MockClient {
	return &MockClient{}
}
