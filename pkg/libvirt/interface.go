package libvirt

import (
	"github.com/digitalocean/go-libvirt"
)

// LibvirtClient 定义 libvirt 客户端接口
// 用于抽象 libvirt 操作，便于测试和 mock
type LibvirtClient interface {
	// 连接信息
	GetHostname() (string, error)
	GetLibvirtVersion() (string, error)
	GetNodeInfo() (*NodeInfo, error)
	GetCapabilities() (string, error)
	GetSysinfo() (string, error)

	// Domain 操作
	GetVMSummaries() ([]libvirt.Domain, error)
	GetDomainInfo(domainUUID libvirt.UUID) (*DomainInfo, error)
	GetDomainByName(name string) (libvirt.Domain, error)
	GetDomainState(domain libvirt.Domain) (uint8, uint32, error)
	CreateDomain(config *CreateVMConfig, autoStart bool) (libvirt.Domain, error)
	StartDomain(domain libvirt.Domain) error
	StopDomain(domain libvirt.Domain) error
	RebootDomain(domain libvirt.Domain) error
	DestroyDomain(domain libvirt.Domain) error
	DeleteDomain(domain libvirt.Domain, flags libvirt.DomainUndefineFlagsValues) error
	ModifyDomainMemory(domain libvirt.Domain, memoryKB uint64, live bool) error
	ModifyDomainVCPU(domain libvirt.Domain, vcpus uint16, live bool) error

	// Domain 磁盘操作
	AttachDiskToDomain(domainName, volumePath, device string) error
	DetachDiskFromDomain(domainName, device string) error
	GetDomainDisks(domainName string) ([]DomainDisk, error)

	// Storage Pool 操作
	GetStoragePool(poolName string) (*StoragePoolInfo, error)
	ListStoragePools() ([]*StoragePoolInfo, error)
	EnsureStoragePool(poolName, poolType, poolPath string) error
	CreateStoragePool(poolName, poolType, poolPath string) error
	StartStoragePool(poolName string) error
	StopStoragePool(poolName string) error
	DeleteStoragePool(poolName string, deleteVolumes bool) error
	RefreshStoragePool(poolName string) error

	// Storage Volume 操作
	GetVolume(poolName, volumeName string) (*VolumeInfo, error)
	ListVolumes(poolName string) ([]*VolumeInfo, error)
	CreateVolume(poolName, volumeName string, sizeGB uint64, format string) (*VolumeInfo, error)
	DeleteVolume(poolName, volumeName string) error

	// QEMU Guest Agent 操作
	QemuAgentCommand(domain libvirt.Domain, command string, timeout uint32, flags uint32) (string, error)
	CheckGuestAgentAvailable(domain libvirt.Domain) (bool, error)

	// Console 操作
	GetDomainConsoleInfo(domain libvirt.Domain) (*ConsoleInfo, error)

	// Snapshot 操作
	ListSnapshots(domainName string) ([]string, error)

	// Network Interface 操作
	ListInterfaces() ([]libvirt.Interface, error)
	GetInterfaceXMLDesc(iface libvirt.Interface) (string, error)

	// Node Device 操作
	ListNodeDevices(cap string) ([]libvirt.NodeDevice, error)
	GetNodeDeviceXMLDesc(dev libvirt.NodeDevice) (string, error)
}
