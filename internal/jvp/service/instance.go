// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service、Template Service 和 Instance Service
package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/cloudinit"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// InstanceService 实例服务，管理虚拟机实例
type InstanceService struct {
	nodeProvider        NodeStorageProvider
	templateService     *TemplateService
	keyPairService      *KeyPairService
	virtCustomizeClient virtcustomize.VirtCustomizeClient
	idGen               *idgen.Generator
	asyncRun            func(func())
	ipCache             map[string]interfaceIPCacheEntry
	ipRefreshInFlight   map[string]struct{}
	ipCacheMu           sync.Mutex
}

type interfaceIPCacheEntry struct {
	IPs       []string
	UpdatedAt time.Time
}

// NodeStorageProvider 定义节点存储获取接口，便于测试替换
type NodeStorageProvider interface {
	GetNodeStorage(ctx context.Context, nodeName string) (libvirt.LibvirtClient, error)
}

// NewInstanceService 创建新的 Instance Service
func NewInstanceService(
	nodeProvider NodeStorageProvider,
	templateService *TemplateService,
	keyPairService *KeyPairService,
) (*InstanceService, error) {
	// 创建 virt-customize 客户端（如果失败，返回 nil，后续使用时再处理）
	virtCustomizeClient, _ := virtcustomize.NewClient()

	return &InstanceService{
		nodeProvider:        nodeProvider,
		templateService:     templateService,
		keyPairService:      keyPairService,
		virtCustomizeClient: virtCustomizeClient,
		idGen:               idgen.New(),
		asyncRun: func(f func()) {
			go f()
		},
		ipCache:           make(map[string]interfaceIPCacheEntry),
		ipRefreshInFlight: make(map[string]struct{}),
	}, nil
}

// GetLibvirtClient 获取指定节点的 libvirt 客户端（用于控制台访问）
func (s *InstanceService) GetLibvirtClient(ctx context.Context, nodeName string) (libvirt.LibvirtClient, error) {
	return s.nodeProvider.GetNodeStorage(ctx, nodeName)
}

// RunInstance 创建并启动实例
func (s *InstanceService) RunInstance(ctx context.Context, req *entity.RunInstanceRequest) (*entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("template_id", req.TemplateID).
		Msg("Creating instance")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 生成实例名称
	instanceName := req.Name
	if instanceName == "" {
		id, err := s.idGen.GenerateID()
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate instance ID", err)
		}
		instanceName = fmt.Sprintf("i-%d", id)
	}

	// 设置默认值
	memoryMB := req.MemoryMB
	if memoryMB == 0 {
		memoryMB = 2048 // 默认 2GB
	}
	vcpus := req.VCPUs
	if vcpus == 0 {
		vcpus = 2 // 默认 2 核
	}
	sizeGB := req.SizeGB
	if sizeGB == 0 {
		sizeGB = 20 // 默认 20GB
	}

	var diskPath string
	var templateID string
	var diskVolumeName string
	var windowsInstallISOPath string
	var windowsDriverISOPath string
	osType := strings.ToLower(strings.TrimSpace(req.OSType))
	if osType == "" {
		osType = "linux"
	}
	isWindows := osType == "windows"
	windowsBootMode := strings.ToLower(strings.TrimSpace(req.WindowsBootMode))
	if isWindows && windowsBootMode == "" {
		windowsBootMode = "install"
	}
	if isWindows && windowsBootMode != "install" && windowsBootMode != "cloud_image" {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "windows_boot_mode must be install or cloud_image", http.StatusBadRequest)
	}
	isWindowsCloudImage := isWindows && windowsBootMode == "cloud_image"
	if isWindows && sizeGB == 20 && req.SizeGB == 0 {
		sizeGB = 64
	}

	// 如果指定了模板，获取模板信息并创建增量磁盘
	if req.TemplateID != "" && (!isWindows || isWindowsCloudImage) {
		// 获取模板信息
		template, err := s.templateService.DescribeTemplate(ctx, &entity.DescribeTemplateRequest{
			NodeName:   req.NodeName,
			PoolName:   req.PoolName,
			TemplateID: req.TemplateID,
		})
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get template", err)
		}

		templateID = template.ID
		if isWindowsCloudImage {
			if !isWindowsTemplate(template) {
				return nil, apierror.NewErrorWithStatus("InvalidParameter", "Windows cloud image template must be identified as Windows", http.StatusBadRequest)
			}
			if !template.Features.CloudInit {
				return nil, apierror.NewErrorWithStatus("InvalidParameter", "Windows cloud image template must have Cloud-init ready enabled", http.StatusBadRequest)
			}
			if !template.Features.Virtio {
				return nil, apierror.NewErrorWithStatus("InvalidParameter", "Windows cloud image template must include VirtIO drivers", http.StatusBadRequest)
			}
			if strings.EqualFold(filepath.Ext(template.VolumeName), ".iso") {
				return nil, apierror.NewErrorWithStatus("InvalidParameter", "Windows cloud image template must be a disk image, not an ISO", http.StatusBadRequest)
			}
		}

		// 使用模板的大小（如果请求没有指定更大的大小）
		if req.SizeGB == 0 || uint64(template.SizeGB) > req.SizeGB {
			sizeGB = uint64(template.SizeGB)
		}
		if sizeGB < uint64(template.SizeGB) {
			sizeGB = uint64(template.SizeGB) // 不能比模板小
		}

		// 创建磁盘卷名称
		diskVolumeName = instanceName + ".qcow2"

		// 使用 backingStore 创建增量磁盘
		logger.Info().
			Str("pool_name", req.PoolName).
			Str("volume_name", diskVolumeName).
			Str("backing_path", template.Path).
			Uint64("size_gb", sizeGB).
			Msg("Creating disk with backing store")

		volumeInfo, err := client.CreateVolumeWithBackingStore(
			req.PoolName,
			diskVolumeName,
			sizeGB,
			"qcow2",
			template.Path,
			template.Format,
		)
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create disk volume", err)
		}
		diskPath = volumeInfo.Path

		logger.Info().
			Str("disk_path", diskPath).
			Msg("Disk volume created")
	} else {
		if isWindows {
			if req.TemplateID == "" {
				message := "template_id is required for Windows install ISO"
				if isWindowsCloudImage {
					message = "template_id is required for Windows cloud image"
				}
				return nil, apierror.NewErrorWithStatus("InvalidParameter", message, http.StatusBadRequest)
			}
			installerTemplate, err := s.templateService.DescribeTemplate(ctx, &entity.DescribeTemplateRequest{
				NodeName:   req.NodeName,
				PoolName:   req.PoolName,
				TemplateID: req.TemplateID,
			})
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get Windows install ISO template", err)
			}
			templateID = installerTemplate.ID
			windowsInstallISOPath = installerTemplate.Path

			if req.DriverISOID != "" {
				driverTemplate, err := s.templateService.DescribeTemplate(ctx, &entity.DescribeTemplateRequest{
					NodeName:   req.NodeName,
					PoolName:   req.PoolName,
					TemplateID: req.DriverISOID,
				})
				if err != nil {
					return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get Windows driver ISO template", err)
				}
				windowsDriverISOPath = driverTemplate.Path
			}
		}

		// 没有模板，创建空白磁盘
		diskVolumeName = instanceName + ".qcow2"

		volumeInfo, err := client.CreateVolume(req.PoolName, diskVolumeName, sizeGB, "qcow2")
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create blank disk volume", err)
		}
		diskPath = volumeInfo.Path
	}

	// 处理 Linux cloud-init 或 Windows Cloudbase-Init 配置
	var cloudInitISOPath string
	needsCloudInit := (!isWindows && (req.UserData != nil || len(req.KeyPairIDs) > 0 || req.TemplateID != "")) || isWindowsCloudImage
	if needsCloudInit {
		var cloudInitConfig *cloudinit.Config
		var userData *cloudinit.UserData
		var err error
		if isWindowsCloudImage {
			cloudInitConfig, userData, err = s.convertWindowsUserDataToCloudInit(instanceName, req.UserData)
		} else {
			cloudInitConfig, userData, err = s.convertUserDataToCloudInit(ctx, instanceName, req.UserData)
		}
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert user data", err)
		}

		if cloudInitConfig == nil && userData == nil {
			cloudInitConfig = &cloudinit.Config{
				Hostname: instanceName,
			}
		}

		// 添加 SSH 密钥。Linux 写入用户配置，Windows NoCloud 写入 meta-data。
		publicKeys := make([]string, 0, len(req.KeyPairIDs))
		if len(req.KeyPairIDs) > 0 && cloudInitConfig != nil {
			for _, keyPairID := range req.KeyPairIDs {
				keyPair, err := s.keyPairService.GetKeyPairByID(ctx, keyPairID)
				if err != nil {
					logger.Warn().
						Str("keypair_id", keyPairID).
						Err(err).
						Msg("Failed to get key pair, skipping")
					continue
				}
				publicKeys = append(publicKeys, keyPair.PublicKey)
				if isWindowsCloudImage {
					if userData != nil && len(userData.Users) > 0 {
						if user, ok := userData.Users[0].(cloudinit.User); ok {
							user.SSHAuthorizedKeys = append(user.SSHAuthorizedKeys, keyPair.PublicKey)
							userData.Users[0] = user
						}
					}
					continue
				}
				// 添加到默认用户的 SSH 密钥
				if len(cloudInitConfig.Users) == 0 {
					cloudInitConfig.Users = []cloudinit.User{{
						Name:              "ubuntu",
						Sudo:              "ALL=(ALL) NOPASSWD:ALL",
						Shell:             "/bin/bash",
						SSHAuthorizedKeys: []string{keyPair.PublicKey},
					}}
				} else {
					cloudInitConfig.Users[0].SSHAuthorizedKeys = append(
						cloudInitConfig.Users[0].SSHAuthorizedKeys,
						keyPair.PublicKey,
					)
				}
			}
		}

		// 设置网络配置
		networkType := req.NetworkType
		if networkType == "" {
			networkType = "bridge"
		}
		networkSource := req.NetworkSource
		if networkSource == "" {
			networkSource = "br0"
		}

		if !isWindowsCloudImage {
			hostBridgeIP := ""
			if networkType == "bridge" {
				hostBridgeIP = lookupBridgeIPv4(client, networkSource)
			}
			ensureQemuGuestAgentCloudInit(cloudInitConfig, userData, hostBridgeIP)
		}

		// 生成 cloud-init ISO
		if cloudInitConfig != nil || userData != nil {
			// 获取存储池路径
			poolInfo, err := client.GetStoragePool(req.PoolName)
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get storage pool", err)
			}

			// 生成 cloud-init 配置文件内容
			generator := cloudinit.NewGenerator()
			metaData, err := generator.GenerateMetaDataWithPublicKeys(cloudInitConfig.Hostname, publicKeys)
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate meta-data", err)
			}

			var userDataContent string
			if userData != nil {
				userDataContent, err = generator.GenerateUserDataFromStruct(userData)
			} else {
				userDataContent, err = generator.GenerateUserData(cloudInitConfig)
			}
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate user-data", err)
			}

			// 在远程节点上生成 cloud-init ISO
			cloudInitISOPath, err = client.CreateCloudInitISO(
				poolInfo.Path,
				instanceName,
				metaData,
				userDataContent,
			)
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate cloud-init ISO on remote node", err)
			}

			logger.Info().
				Str("cloud_init_iso", cloudInitISOPath).
				Msg("Cloud-init ISO generated on remote node")
		}
	}

	// 设置网络配置
	networkType := req.NetworkType
	if networkType == "" {
		networkType = "bridge"
	}
	networkSource := req.NetworkSource
	if networkSource == "" {
		networkSource = "br0"
	}

	// 创建 Domain
	vmConfig := &libvirt.CreateVMConfig{
		Name:           instanceName,
		Memory:         memoryMB * 1024, // 转换为 KB
		VCPUs:          vcpus,
		DiskPath:       diskPath,
		NetworkType:    networkType,
		NetworkSource:  networkSource,
		QemuGuestAgent: true,
	}

	if isWindows {
		vmConfig.MachineType = "q35"
		vmConfig.DiskBus = "sata"
		vmConfig.CPUMode = "host-passthrough"
		vmConfig.CPUSockets = 1
		vmConfig.CPUCores = int(vcpus)
		vmConfig.CPUThreads = 1
		vmConfig.LoaderPath = "/usr/share/OVMF/OVMF_CODE_4M.ms.fd"
		vmConfig.NVRAMTemplate = "/usr/share/OVMF/OVMF_VARS_4M.ms.fd"
		vmConfig.NVRAMPath = "/var/lib/libvirt/qemu/nvram/" + instanceName + "_VARS.fd"
		vmConfig.TPM = true
		vmConfig.SMM = true
		if isWindowsCloudImage {
			if cloudInitISOPath != "" {
				vmConfig.CDROMs = append(vmConfig.CDROMs, libvirt.CDROMConfig{
					Path:   cloudInitISOPath,
					Device: "sdb",
					Bus:    "sata",
				})
			}
		} else {
			vmConfig.DisableOSBoot = true
			vmConfig.CDROMs = append(vmConfig.CDROMs, libvirt.CDROMConfig{
				Path:   windowsInstallISOPath,
				Device: "sdb",
				Bus:    "sata",
				Boot:   true,
			})
			if windowsDriverISOPath != "" {
				vmConfig.CDROMs = append(vmConfig.CDROMs, libvirt.CDROMConfig{
					Path:   windowsDriverISOPath,
					Device: "sdc",
					Bus:    "sata",
				})
			}
		}
	}

	// 如果有 cloud-init ISO，添加到配置
	if cloudInitISOPath != "" && !isWindowsCloudImage {
		vmConfig.ISOPath = cloudInitISOPath
	}

	logger.Info().
		Str("name", instanceName).
		Uint64("memory_mb", memoryMB).
		Uint16("vcpus", vcpus).
		Str("disk_path", diskPath).
		Msg("Creating domain")

	domain, err := client.CreateDomain(vmConfig, true)
	if err != nil {
		if diskVolumeName != "" {
			if cleanupErr := client.DeleteVolume(req.PoolName, diskVolumeName); cleanupErr != nil {
				logger.Warn().
					Err(cleanupErr).
					Str("pool_name", req.PoolName).
					Str("volume_name", diskVolumeName).
					Msg("Failed to clean up disk volume after domain creation failure")
			}
		}
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create domain", err)
	}

	// 启动 domain
	if err := client.StartDomain(domain); err != nil {
		logger.Warn().
			Err(err).
			Str("name", instanceName).
			Msg("Failed to start domain, it might already be running")
	}
	if isWindows && !isWindowsCloudImage {
		go s.sendWindowsInstallerBootKey(client, domain, instanceName, logger.With().Logger())
	}

	logger.Info().
		Str("name", instanceName).
		Str("domain_uuid", formatDomainUUID(domain.UUID)).
		Msg("Instance created successfully")

	return &entity.Instance{
		ID:         instanceName,
		Name:       instanceName,
		State:      "running",
		NodeName:   req.NodeName,
		TemplateID: templateID,
		MemoryMB:   memoryMB,
		VCPUs:      vcpus,
		CreatedAt:  time.Now().Format(time.RFC3339),
		DomainUUID: formatDomainUUID(domain.UUID),
		DomainName: instanceName,
	}, nil
}

func (s *InstanceService) sendWindowsInstallerBootKey(client libvirt.LibvirtClient, domain libvirtlib.Domain, instanceName string, logger zerolog.Logger) {
	spaceKey := []uint32{0x39}
	for _, delay := range []time.Duration{2 * time.Second, 4 * time.Second} {
		time.Sleep(delay)
		if err := client.SendKey(domain, uint32(libvirtlib.KeycodeSetXt), spaceKey); err != nil {
			logger.Warn().
				Err(err).
				Str("name", instanceName).
				Msg("Failed to send Windows installer boot key")
			return
		}
	}
	logger.Debug().
		Str("name", instanceName).
		Msg("Sent Windows installer boot key")
}

func (s *InstanceService) shouldSendWindowsInstallerBootKey(client libvirt.LibvirtClient, domainName string, logger zerolog.Logger) bool {
	disks, err := client.GetDomainDisks(domainName)
	if err != nil {
		logger.Debug().
			Err(err).
			Str("name", domainName).
			Msg("Failed to inspect domain disks for Windows installer boot key")
		return false
	}

	for _, disk := range disks {
		if disk.Device != "cdrom" || disk.Boot == nil || disk.Boot.Order != 1 {
			continue
		}

		source := strings.ToLower(disk.Source.File)
		if strings.Contains(source, "windows") || strings.Contains(source, "win11") || strings.Contains(source, "win10") {
			return true
		}
	}

	return false
}

// formatDomainUUID 格式化 Domain UUID
func formatDomainUUID(uuid [16]byte) string {
	return hex.EncodeToString(uuid[:])
}

func isWindowsTemplate(template *entity.Template) bool {
	if template == nil {
		return false
	}
	metadata := strings.ToLower(strings.Join([]string{
		template.Name,
		template.VolumeName,
		template.OS.Name,
		strings.Join(template.Tags, " "),
	}, " "))
	return strings.Contains(metadata, "windows") || strings.Contains(metadata, "win10") || strings.Contains(metadata, "win11")
}

// convertWindowsUserDataToCloudInit maps the shared API shape to the subset
// supported by Cloudbase-Init. Windows passwords must remain plaintext here.
func (s *InstanceService) convertWindowsUserDataToCloudInit(
	instanceID string,
	userDataConfig *entity.UserDataConfig,
) (*cloudinit.Config, *cloudinit.UserData, error) {
	config := &cloudinit.Config{Hostname: instanceID}
	if userDataConfig == nil {
		return config, &cloudinit.UserData{}, nil
	}

	if rawUserData := strings.TrimSpace(userDataConfig.RawUserData); rawUserData != "" {
		config.CustomUserData = rawUserData
		return config, nil, nil
	}
	if userDataConfig.StructuredUserData == nil {
		return config, &cloudinit.UserData{}, nil
	}

	structured := userDataConfig.StructuredUserData
	if structured.Hostname != "" {
		config.Hostname = structured.Hostname
	}
	userData := &cloudinit.UserData{
		SetTimezone: structured.Timezone,
		RunCmd:      append([]string(nil), structured.RunCmd...),
	}

	if len(structured.Groups) > 0 {
		userData.Groups = make(map[string][]string, len(structured.Groups))
		for _, group := range structured.Groups {
			userData.Groups[group.Name] = append([]string(nil), group.Members...)
		}
	}
	for _, sourceUser := range structured.Users {
		if sourceUser.HashedPasswd != "" {
			return nil, nil, fmt.Errorf("hashed_passwd is not supported by Cloudbase-Init; use plain_text_passwd")
		}
		userData.Users = append(userData.Users, cloudinit.User{
			Name:              sourceUser.Name,
			Groups:            sourceUser.Groups,
			Passwd:            sourceUser.PlainTextPasswd,
			SSHAuthorizedKeys: append([]string(nil), sourceUser.SSHAuthorizedKeys...),
		})
	}
	for _, sourceFile := range structured.WriteFiles {
		userData.WriteFiles = append(userData.WriteFiles, cloudinit.WriteFile{
			Path:        sourceFile.Path,
			Content:     sourceFile.Content,
			Permissions: sourceFile.Permissions,
		})
	}

	return config, userData, nil
}

// convertUserDataToCloudInit 将 entity.UserDataConfig 转换为 cloudinit 配置
func (s *InstanceService) convertUserDataToCloudInit(
	ctx context.Context,
	instanceID string,
	userDataConfig *entity.UserDataConfig,
) (*cloudinit.Config, *cloudinit.UserData, error) {
	if userDataConfig == nil {
		return nil, nil, nil
	}

	// 如果提供了原始 YAML，直接使用 UserData
	if userDataConfig.RawUserData != "" {
		// 移除可能的 #cloud-config header（如果存在）
		rawData := strings.TrimSpace(userDataConfig.RawUserData)
		if strings.HasPrefix(rawData, "#cloud-config") {
			// 移除 header
			lines := strings.Split(rawData, "\n")
			if len(lines) > 1 {
				rawData = strings.Join(lines[1:], "\n")
			} else {
				rawData = ""
			}
		}

		// 解析为 UserData 结构
		var userData cloudinit.UserData
		if err := yaml.Unmarshal([]byte(rawData), &userData); err != nil {
			return nil, nil, fmt.Errorf("invalid raw userdata YAML: %w", err)
		}

		// 创建最小的 Config（仅用于 hostname）
		config := &cloudinit.Config{
			Hostname: instanceID,
		}

		return config, &userData, nil
	}

	// 使用结构化配置
	if userDataConfig.StructuredUserData == nil {
		return nil, nil, nil
	}

	structured := userDataConfig.StructuredUserData

	// 转换为 cloudinit.Config
	config := &cloudinit.Config{
		Hostname:    structured.Hostname,
		DisableRoot: structured.DisableRoot,
		Timezone:    structured.Timezone,
		Packages:    structured.Packages,
		Commands:    structured.RunCmd,
	}

	// 转换 Groups
	if len(structured.Groups) > 0 {
		config.Groups = make([]cloudinit.Group, len(structured.Groups))
		for i, g := range structured.Groups {
			config.Groups[i] = cloudinit.Group{
				Name:    g.Name,
				Members: g.Members,
			}
		}
	}

	// 转换 Users
	if len(structured.Users) > 0 {
		config.Users = make([]cloudinit.User, len(structured.Users))
		for i, u := range structured.Users {
			user := cloudinit.User{
				Name:              u.Name,
				Groups:            u.Groups,
				SSHAuthorizedKeys: u.SSHAuthorizedKeys,
				Shell:             u.Shell,
			}

			// 处理密码
			if u.HashedPasswd != "" {
				user.Passwd = u.HashedPasswd
			} else if u.PlainTextPasswd != "" {
				// 如果提供了明文密码，需要 hash
				hashed, err := cloudinit.HashPassword(u.PlainTextPasswd)
				if err != nil {
					return nil, nil, fmt.Errorf("hash password for user %s: %w", u.Name, err)
				}
				user.Passwd = hashed
			}

			// 处理 sudo
			if u.Sudo != "" {
				user.Sudo = u.Sudo
			}

			config.Users[i] = user
		}
	}

	// 转换 WriteFiles
	if len(structured.WriteFiles) > 0 {
		config.WriteFiles = make([]cloudinit.File, len(structured.WriteFiles))
		for i, f := range structured.WriteFiles {
			config.WriteFiles[i] = cloudinit.File{
				Path:        f.Path,
				Content:     f.Content,
				Owner:       f.Owner,
				Permissions: f.Permissions,
			}
		}
	}

	// 设置默认 hostname（如果未设置）
	if config.Hostname == "" {
		config.Hostname = instanceID
	}

	return config, nil, nil
}

// DescribeInstances 描述实例
func (s *InstanceService) DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Msg("Describing instances from libvirt")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 直接从 libvirt 获取所有 domain
	domains, err := client.GetVMSummaries()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get VMs from libvirt")
		return nil, fmt.Errorf("get VMs from libvirt: %w", err)
	}

	logger.Debug().
		Int("total_domains", len(domains)).
		Msg("Retrieved domains from libvirt")

	// 转换为 Instance 对象
	instances := make([]entity.Instance, 0, len(domains))
	for _, domain := range domains {
		// 获取详细信息
		domainInfo, err := client.GetDomainInfo(domain.UUID)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to get domain info, skipping")
			continue
		}

		// 获取状态
		state, _, err := client.GetDomainState(domain)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to get domain state")
			state = 5 // Unknown
		}

		instanceState := convertDomainState(state)
		interfaces := s.convertInterfaces(ctx, client, req.NodeName, domain, domainInfo.NetworkInfo, instanceState)
		instance := entity.Instance{
			ID:         domain.Name, // 使用 domain name 作为 ID
			Name:       domain.Name,
			State:      instanceState,
			NodeName:   req.NodeName,
			DomainUUID: formatDomainUUID(domain.UUID),
			DomainName: domain.Name,
			VCPUs:      domainInfo.VCPUs,
			MemoryMB:   domainInfo.Memory / 1024, // 转换为 MB
			CreatedAt:  "",                       // libvirt 不提供创建时间
			Autostart:  domainInfo.Autostart,
			IPAddress:  primaryIPFromInterfaces(interfaces),
			Interfaces: interfaces,
			StartedAt:  formatStartTime(domainInfo.StartTime),
			Disks:      convertDisks(client, domain.Name),
		}

		// TODO: 如果需要 TemplateID，可以从 domain metadata 读取
		instances = append(instances, instance)
	}

	// 应用过滤器（如果需要）
	instances = s.applyInstanceFilters(instances, req)

	// 按 name 升序排序
	s.sortInstancesByName(instances)

	logger.Info().
		Int("total", len(instances)).
		Msg("Describe instances completed")

	return instances, nil
}

// applyInstanceFilters 应用过滤器
func (s *InstanceService) applyInstanceFilters(instances []entity.Instance, req *entity.DescribeInstancesRequest) []entity.Instance {
	if req == nil {
		return instances
	}

	// 如果指定了 InstanceIDs，只返回匹配的
	if len(req.InstanceIDs) > 0 {
		filtered := make([]entity.Instance, 0)
		idSet := make(map[string]bool)
		for _, id := range req.InstanceIDs {
			idSet[id] = true
		}
		for _, instance := range instances {
			if idSet[instance.ID] {
				filtered = append(filtered, instance)
			}
		}
		return filtered
	}

	return instances
}

// convertDomainState 转换 libvirt 状态为 JVP 状态
func convertDomainState(state uint8) string {
	switch state {
	case 1: // Running
		return "running"
	case 3: // Paused
		return "stopped"
	case 4: // Shutdown
		return "stopped"
	case 5: // Shutoff
		return "stopped"
	case 6: // Crashed
		return "failed"
	default:
		return "pending"
	}
}

func (s *InstanceService) convertInterfaces(
	ctx context.Context,
	client libvirt.LibvirtClient,
	nodeName string,
	domain libvirtlib.Domain,
	ifaces []libvirt.NetworkInterface,
	instanceState string,
) []entity.InstanceInterface {
	result := make([]entity.InstanceInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		key := interfaceIPCacheKey(nodeName, domain.Name, iface.MAC, iface.Source)
		ips, refreshDue := s.cachedInterfaceIPs(key)
		if instanceState == "running" && refreshDue {
			s.refreshInterfaceIPsAsync(ctx, client, domain, iface.MAC, iface.Source, key)
		}
		result = append(result, entity.InstanceInterface{
			Name:   iface.Name,
			Type:   iface.Type,
			Source: iface.Source,
			MAC:    iface.MAC,
			IPs:    ips,
		})
	}
	return result
}

func interfaceIPCacheKey(nodeName, domainName, mac, source string) string {
	return strings.Join([]string{nodeName, domainName, strings.ToLower(mac), source}, "|")
}

func (s *InstanceService) cachedInterfaceIPs(key string) ([]string, bool) {
	const refreshAfter = 30 * time.Second

	s.ipCacheMu.Lock()
	defer s.ipCacheMu.Unlock()

	entry, ok := s.ipCache[key]
	if !ok {
		return nil, true
	}

	ips := append([]string(nil), entry.IPs...)
	return ips, time.Since(entry.UpdatedAt) > refreshAfter
}

func (s *InstanceService) refreshInterfaceIPsAsync(
	ctx context.Context,
	client libvirt.LibvirtClient,
	domain libvirtlib.Domain,
	mac string,
	source string,
	key string,
) {
	s.ipCacheMu.Lock()
	if _, ok := s.ipRefreshInFlight[key]; ok {
		s.ipCacheMu.Unlock()
		return
	}
	s.ipRefreshInFlight[key] = struct{}{}
	s.ipCacheMu.Unlock()

	logger := zerolog.Ctx(ctx).With().
		Str("domain", domain.Name).
		Str("mac", mac).
		Str("source", source).
		Logger()

	s.asyncRun(func() {
		defer func() {
			s.ipCacheMu.Lock()
			delete(s.ipRefreshInFlight, key)
			s.ipCacheMu.Unlock()
		}()

		ips := resolveInterfaceIPs(client, domain, mac, source)
		s.ipCacheMu.Lock()
		s.ipCache[key] = interfaceIPCacheEntry{
			IPs:       append([]string(nil), ips...),
			UpdatedAt: time.Now(),
		}
		s.ipCacheMu.Unlock()

		logger.Debug().
			Strs("ips", ips).
			Msg("Refreshed instance interface IP cache")
	})
}

func resolveInterfaceIPs(client libvirt.LibvirtClient, domain libvirtlib.Domain, mac string, source string) []string {
	var ips []string
	if qgaIPs, err := resolveIPsByGuestAgent(client, domain, mac); err == nil {
		ips = append(ips, qgaIPs...)
	}

	if fallbackIPs, err := libvirt.ResolveIPsByMACOnInterface(client, mac, source); err == nil {
		ips = append(ips, fallbackIPs...)
	}

	return normalizeInstanceIPs(ips)
}

func resolveIPsByGuestAgent(client libvirt.LibvirtClient, domain libvirtlib.Domain, mac string) ([]string, error) {
	if domain.Name == "" || mac == "" {
		return nil, nil
	}

	const command = `{"execute":"guest-network-get-interfaces"}`
	output, err := client.QemuAgentCommand(domain, command, 5, 0)
	if err != nil {
		return nil, err
	}

	var response struct {
		Return []struct {
			HardwareAddress string `json:"hardware-address"`
			IPAddresses     []struct {
				IPAddress string `json:"ip-address"`
			} `json:"ip-addresses"`
		} `json:"return"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, err
	}

	mac = strings.ToLower(mac)
	var ips []string
	for _, iface := range response.Return {
		if strings.ToLower(iface.HardwareAddress) != mac {
			continue
		}
		for _, addr := range iface.IPAddresses {
			ips = append(ips, addr.IPAddress)
		}
	}

	return ips, nil
}

func normalizeInstanceIPs(ips []string) []string {
	seen := make(map[string]struct{}, len(ips))
	var publicOrPrivate []string
	var linkLocal []string

	for _, ip := range ips {
		addr, err := netip.ParseAddr(strings.TrimSpace(ip))
		if err != nil || addr.IsLoopback() || addr.IsUnspecified() {
			continue
		}

		normalized := addr.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}

		if addr.IsLinkLocalUnicast() {
			linkLocal = append(linkLocal, normalized)
			continue
		}
		publicOrPrivate = append(publicOrPrivate, normalized)
	}

	sort.SliceStable(publicOrPrivate, func(i, j int) bool {
		left, right := netip.MustParseAddr(publicOrPrivate[i]), netip.MustParseAddr(publicOrPrivate[j])
		if left.Is4() != right.Is4() {
			return left.Is4()
		}
		return publicOrPrivate[i] < publicOrPrivate[j]
	})
	sort.Strings(linkLocal)

	if len(publicOrPrivate) > 0 {
		return publicOrPrivate
	}
	return linkLocal
}

func primaryIPFromInterfaces(ifaces []entity.InstanceInterface) string {
	for _, iface := range ifaces {
		for _, ip := range iface.IPs {
			addr, err := netip.ParseAddr(ip)
			if err == nil && addr.Is4() {
				return ip
			}
		}
	}
	for _, iface := range ifaces {
		if len(iface.IPs) > 0 {
			return iface.IPs[0]
		}
	}
	return ""
}

func ensureQemuGuestAgentCloudInit(config *cloudinit.Config, userData *cloudinit.UserData, hostBridgeIP string) {
	commands := []string{
		"( if ! command -v qemu-ga >/dev/null 2>&1 && ! command -v qemu-guest-agent >/dev/null 2>&1; then if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; timeout 600 apt-get update -o Acquire::Retries=3 -o Acquire::Languages=none -o Acquire::IndexTargets::deb-src::Sources::DefaultEnabled=false && timeout 300 apt-get install -y --no-install-recommends qemu-guest-agent; elif command -v dnf >/dev/null 2>&1; then timeout 300 dnf install -y qemu-guest-agent; elif command -v yum >/dev/null 2>&1; then timeout 300 yum install -y qemu-guest-agent; elif command -v zypper >/dev/null 2>&1; then timeout 300 zypper --non-interactive install qemu-guest-agent; elif command -v apk >/dev/null 2>&1; then timeout 300 apk add --no-cache qemu-guest-agent; elif command -v pacman >/dev/null 2>&1; then timeout 300 pacman -Sy --noconfirm qemu-guest-agent; fi; fi; if command -v systemctl >/dev/null 2>&1; then systemctl enable --now qemu-guest-agent || systemctl enable --now qemu-ga || true; elif command -v rc-update >/dev/null 2>&1; then rc-update add qemu-guest-agent default || true; rc-service qemu-guest-agent start || true; fi ) >/var/log/jvp-qga-bootstrap.log 2>&1 &",
	}
	if hostBridgeIP != "" {
		commands = append([]string{buildNeighborRefreshCommand(hostBridgeIP)}, commands...)
	}

	if config != nil {
		for _, command := range commands {
			config.Commands = appendUniqueString(config.Commands, command)
		}
	}
	if userData != nil {
		for _, command := range commands {
			userData.RunCmd = appendUniqueString(userData.RunCmd, command)
		}
	}
}

func buildNeighborRefreshCommand(hostBridgeIP string) string {
	pingLoop := fmt.Sprintf("while true; do ping -c 1 -W 1 %s >/dev/null 2>&1 || true; sleep 60; done", hostBridgeIP)
	return fmt.Sprintf("if command -v systemctl >/dev/null 2>&1; then cat >/etc/systemd/system/jvp-neighbor-refresh.service <<'EOF'\n[Unit]\nDescription=Refresh JVP host neighbor entry\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\nType=simple\nExecStart=/bin/sh -c '%s'\nRestart=always\nRestartSec=10\n\n[Install]\nWantedBy=multi-user.target\nEOF\nsystemctl daemon-reload && systemctl enable --now jvp-neighbor-refresh.service || true; else ( %s ) >/var/log/jvp-neighbor-refresh.log 2>&1 & fi", pingLoop, pingLoop)
}

func lookupBridgeIPv4(client libvirt.LibvirtClient, bridgeName string) string {
	if bridgeName == "" {
		return ""
	}
	if client.IsRemoteConnection() {
		const outputPath = "/tmp/.jvp_bridge_ipv4"
		command := fmt.Sprintf("ip -4 -o addr show dev %s scope global | awk '{print $4}' | head -n1 > %s", shellQuote(bridgeName), outputPath)
		if err := client.ExecuteRemoteCommand(command); err != nil {
			return ""
		}
		data, err := client.ReadRemoteFile(outputPath)
		if err != nil {
			return ""
		}
		return firstIPv4FromCIDROutput(string(data))
	}

	out, err := exec.Command("ip", "-4", "-o", "addr", "show", "dev", bridgeName, "scope", "global").Output()
	if err != nil {
		return ""
	}
	return firstIPv4FromAddrShow(string(out))
}

func firstIPv4FromAddrShow(output string) string {
	for _, field := range strings.Fields(output) {
		if ip := firstIPv4FromCIDROutput(field); ip != "" {
			return ip
		}
	}
	return ""
}

func firstIPv4FromCIDROutput(output string) string {
	for _, field := range strings.Fields(output) {
		prefix, err := netip.ParsePrefix(field)
		if err != nil || !prefix.Addr().Is4() || prefix.Addr().IsLoopback() {
			continue
		}
		return prefix.Addr().String()
	}
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func formatStartTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func convertDisks(client libvirt.LibvirtClient, domainName string) []entity.InstanceDisk {
	disks, err := client.GetDomainDisks(domainName)
	if err != nil {
		return nil
	}
	result := make([]entity.InstanceDisk, 0, len(disks))
	for _, d := range disks {
		result = append(result, entity.InstanceDisk{
			Target:      d.Target.Dev,
			Path:        d.Source.File,
			Format:      d.Driver.Type,
			Device:      d.Device,
			CapacityB:   d.CapacityB,
			AllocationB: d.AllocationB,
		})
	}
	return result
}

// EjectInstanceMedia ejects removable media from an instance CD-ROM device.
func (s *InstanceService) EjectInstanceMedia(ctx context.Context, req *entity.EjectInstanceMediaRequest) (*entity.Instance, error) {
	if req == nil {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "request body is required", 400)
	}
	target := strings.TrimSpace(req.Target)
	if target == "" {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "target is required", 400)
	}

	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	disks, err := client.GetDomainDisks(req.InstanceID)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain disks", err)
	}

	found := false
	for _, disk := range disks {
		if disk.Target.Dev != target {
			continue
		}
		found = true
		if disk.Device != "cdrom" {
			return nil, apierror.NewErrorWithStatus("InvalidParameter", "only CD-ROM media can be ejected", 400)
		}
		break
	}
	if !found {
		return nil, apierror.NewErrorWithStatus("ResourceNotFound", "disk target not found", 404)
	}

	if err := client.EjectMediaFromDomain(req.InstanceID, target); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to eject media", err)
	}

	return s.GetInstance(ctx, req.NodeName, req.InstanceID)
}

// GetInstance 获取单个实例信息
func (s *InstanceService) GetInstance(ctx context.Context, nodeName, instanceID string) (*entity.Instance, error) {
	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, nodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// instanceID 就是 domain name
	domain, err := client.GetDomainByName(instanceID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// 获取详细信息
	domainInfo, err := client.GetDomainInfo(domain.UUID)
	if err != nil {
		return nil, fmt.Errorf("get domain info: %w", err)
	}

	// 获取状态
	state, _, err := client.GetDomainState(domain)
	if err != nil {
		state = 5 // Unknown
	}

	instanceState := convertDomainState(state)
	interfaces := s.convertInterfaces(ctx, client, nodeName, domain, domainInfo.NetworkInfo, instanceState)
	instance := &entity.Instance{
		ID:         domain.Name,
		Name:       domain.Name,
		State:      instanceState,
		NodeName:   nodeName,
		DomainUUID: formatDomainUUID(domain.UUID),
		DomainName: domain.Name,
		VCPUs:      domainInfo.VCPUs,
		MemoryMB:   domainInfo.Memory / 1024,
		CreatedAt:  time.Now().Format(time.RFC3339),
		Autostart:  domainInfo.Autostart,
		IPAddress:  primaryIPFromInterfaces(interfaces),
		Interfaces: interfaces,
		StartedAt:  formatStartTime(domainInfo.StartTime),
		Disks:      convertDisks(client, domain.Name),
	}

	return instance, nil
}

// TerminateInstances 终止实例
func (s *InstanceService) TerminateInstances(ctx context.Context, req *entity.TerminateInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Terminating instances")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, req.NodeName, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		// 获取 domain
		domain, err := client.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 记录磁盘路径，用于后续删除卷
		disks, err := client.GetDomainDisks(instanceID)
		if err != nil {
			logger.Warn().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain disks before deletion")
		}

		// 删除 domain（会先停止运行中的实例）
		// 使用 DomainUndefineSnapshotsMetadata 标志同时删除快照元数据
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Deleting domain from libvirt")
		undefineFlags := libvirtlib.DomainUndefineSnapshotsMetadata | libvirtlib.DomainUndefineNvram
		err = client.DeleteDomain(domain, libvirtlib.DomainUndefineFlagsValues(undefineFlags))
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to delete domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to delete domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain deleted successfully")

		// 删除关联的卷（可选）
		if req.DeleteVolumes && len(disks) > 0 {
			if err := s.deleteVolumesByDisks(ctx, client, disks); err != nil {
				logger.Error().
					Str("instanceID", instanceID).
					Err(err).
					Msg("Failed to delete volumes for instance")
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to delete instance volumes", err)
			}
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Associated volumes deleted")
		}

		// Domain 已从 libvirt 删除，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "terminated",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "terminated").
			Msg("Instance terminated successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to terminate")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances terminated successfully")

	return changes, nil
}

// deleteVolumesByDisks 根据磁盘列表删除对应的卷（优先按路径删除）
func (s *InstanceService) deleteVolumesByDisks(ctx context.Context, client libvirt.LibvirtClient, disks []libvirt.DomainDisk) error {
	logger := zerolog.Ctx(ctx)

	for _, disk := range disks {
		if disk.Source.File == "" {
			continue
		}
		if disk.Device != "disk" {
			logger.Debug().
				Str("path", disk.Source.File).
				Str("device", disk.Device).
				Msg("Skipping non-disk device during volume deletion")
			continue
		}
		if strings.Contains(disk.Source.File, "/_templates_/") {
			logger.Warn().
				Str("path", disk.Source.File).
				Msg("Skipping template volume during instance volume deletion")
			continue
		}

		// 优先使用 libvirt 路径删除
		if err := client.DeleteVolumeByPath(disk.Source.File); err != nil {
			// 如果 libvirt 删除失败（可能不在存储池中），尝试直接删除文件
			logger.Debug().
				Err(err).
				Str("path", disk.Source.File).
				Msg("Failed to delete volume via libvirt, trying direct file deletion")

			if client.IsRemoteConnection() {
				// 远程连接：通过 SSH 删除
				if rmErr := client.ExecuteRemoteCommand(fmt.Sprintf("rm -f '%s'", disk.Source.File)); rmErr != nil {
					logger.Warn().
						Err(rmErr).
						Str("path", disk.Source.File).
						Msg("Failed to delete volume file remotely, skipping")
				} else {
					logger.Info().
						Str("path", disk.Source.File).
						Msg("Deleted instance volume file remotely")
				}
			} else {
				// 本地连接：直接删除文件
				if rmErr := os.Remove(disk.Source.File); rmErr != nil && !os.IsNotExist(rmErr) {
					logger.Warn().
						Err(rmErr).
						Str("path", disk.Source.File).
						Msg("Failed to delete volume file, skipping")
				} else {
					logger.Info().
						Str("path", disk.Source.File).
						Msg("Deleted instance volume file")
				}
			}
		} else {
			logger.Info().
				Str("path", disk.Source.File).
				Msg("Deleted instance volume")
		}
	}

	return nil
}

// StopInstances 停止实例
func (s *InstanceService) StopInstances(ctx context.Context, req *entity.StopInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Strs("instanceIDs", req.InstanceIDs).
		Bool("force", req.Force).
		Msg("Stopping instances")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, req.NodeName, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 已经停止，跳过
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance already stopped, skipping")
			changes = append(changes, entity.InstanceStateChange{
				InstanceID:    instanceID,
				CurrentState:  "stopped",
				PreviousState: previousState,
			})
			continue
		}

		// 获取 domain
		domain, err := client.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 停止 domain
		if req.Force {
			// 强制停止
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Force stopping domain")
			err = client.DestroyDomain(domain)
		} else {
			// 优雅停止
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Gracefully stopping domain")
			err = client.StopDomain(domain)
		}

		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Bool("force", req.Force).
				Err(err).
				Msg("Failed to stop domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to stop domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain stop command sent successfully")

		// 状态已在 libvirt 中更新，不需要额外操作

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "stopped",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "stopped").
			Msg("Instance stopped successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to stop")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances stopped successfully")

	return changes, nil
}

// StartInstances 启动实例
func (s *InstanceService) StartInstances(ctx context.Context, req *entity.StartInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Starting instances")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, req.NodeName, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "running" {
			// 已经运行，跳过
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance already running, skipping")
			changes = append(changes, entity.InstanceStateChange{
				InstanceID:    instanceID,
				CurrentState:  "running",
				PreviousState: previousState,
			})
			continue
		}

		// 获取 domain
		domain, err := client.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 启动 domain
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Starting domain")
		err = client.StartDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to start domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to start domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain start command sent successfully")
		if s.shouldSendWindowsInstallerBootKey(client, instanceID, logger.With().Logger()) {
			go s.sendWindowsInstallerBootKey(client, domain, instanceID, logger.With().Logger())
		}

		// 状态已在 libvirt 中更新，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "running").
			Msg("Instance started successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to start")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances started successfully")

	return changes, nil
}

// RebootInstances 重启实例
func (s *InstanceService) RebootInstances(ctx context.Context, req *entity.RebootInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Rebooting instances")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, req.NodeName, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 如果已停止，先启动
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance is stopped, starting before reboot")
			_, err = s.StartInstances(ctx, &entity.StartInstancesRequest{
				NodeName:    req.NodeName,
				InstanceIDs: []string{instanceID},
			})
			if err != nil {
				logger.Error().
					Str("instanceID", instanceID).
					Err(err).
					Msg("Failed to start instance before reboot")
				lastError = err
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to start instance before reboot", err)
			}
			previousState = "running"
		}

		// 获取 domain
		domain, err := client.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 重启 domain
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Rebooting domain")
		err = client.RebootDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to reboot domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to reboot domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain reboot command sent successfully")
		if s.shouldSendWindowsInstallerBootKey(client, instanceID, logger.With().Logger()) {
			go s.sendWindowsInstallerBootKey(client, domain, instanceID, logger.With().Logger())
		}

		// 状态已在 libvirt 中更新，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "running").
			Msg("Instance rebooted successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to reboot")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances rebooted successfully")

	return changes, nil
}

// ModifyInstanceAttribute 修改实例属性
func (s *InstanceService) ModifyInstanceAttribute(ctx context.Context, req *entity.ModifyInstanceAttributeRequest) (*entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("instanceID", req.InstanceID).
		Interface("request", req).
		Msg("Modifying instance attribute")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 获取当前实例信息
	instance, err := s.GetInstance(ctx, req.NodeName, req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	// 获取 domain
	domain, err := client.GetDomainByName(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}

	// 修改内存
	if req.MemoryMB != nil {
		memoryKB := *req.MemoryMB * 1024
		err = client.ModifyDomainMemory(domain, memoryKB, req.Live)
		if err != nil {
			return nil, fmt.Errorf("modify memory: %w", err)
		}
		instance.MemoryMB = *req.MemoryMB
		logger.Info().
			Str("instanceID", req.InstanceID).
			Uint64("memoryMB", *req.MemoryMB).
			Msg("Instance memory modified")
	}

	// 修改 VCPU
	if req.VCPUs != nil {
		err = client.ModifyDomainVCPU(domain, *req.VCPUs, req.Live)
		if err != nil {
			return nil, fmt.Errorf("modify VCPU: %w", err)
		}
		instance.VCPUs = *req.VCPUs
		logger.Info().
			Str("instanceID", req.InstanceID).
			Uint16("vcpus", *req.VCPUs).
			Msg("Instance VCPU modified")
	}

	// 修改名称（TODO: 需要更新 domain XML 的 name 字段）
	if req.Name != nil {
		// TODO: 实现修改 domain 名称
		instance.Name = *req.Name
		logger.Info().
			Str("instanceID", req.InstanceID).
			Str("name", *req.Name).
			Msg("Instance name modified")
	}

	// 修改自动启动
	if req.Autostart != nil {
		err = client.SetDomainAutostart(domain, *req.Autostart)
		if err != nil {
			return nil, fmt.Errorf("modify autostart: %w", err)
		}
		instance.Autostart = *req.Autostart
		logger.Info().
			Str("instanceID", req.InstanceID).
			Bool("autostart", *req.Autostart).
			Msg("Instance autostart modified")
	}

	// 属性已在 libvirt 中更新，重新获取实例信息以获取最新状态
	updatedInstance, err := s.GetInstance(ctx, req.NodeName, req.InstanceID)
	if err != nil {
		// 如果获取失败，返回修改后的实例信息
		return instance, nil
	}

	logger.Info().
		Str("instanceID", req.InstanceID).
		Msg("Instance attribute modified successfully")

	return updatedInstance, nil
}

// ResetPassword 重置实例密码，异步执行：
// 1. qemu-guest-agent（优先，不需要停止实例）
// 2. cloud-init（失败则回退）
// 3. virt-customize（最后选择，远程节点通过 SSH 调用）
func (s *InstanceService) ResetPassword(ctx context.Context, req *entity.ResetPasswordRequest) (*entity.ResetPasswordResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("instance_id", req.InstanceID).
		Int("user_count", len(req.Users)).
		Msg("Resetting instance password")

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 1. 验证实例存在
	instance, err := s.GetInstance(ctx, req.NodeName, req.InstanceID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("Instance %s not found", req.InstanceID),
			404,
		)
	}

	// 2. 构建用户密码映射
	usersMap := make(map[string]string)
	userList := make([]string, 0, len(req.Users))
	for _, user := range req.Users {
		usersMap[user.Username] = user.NewPassword
		userList = append(userList, user.Username)
	}

	// 3. 异步执行重置
	wasRunning := instance.State == "running"
	reqCopy := *req
	ctxCopy := context.WithoutCancel(ctx)
	s.asyncRun(func() {
		logger := zerolog.Ctx(ctxCopy)

		stoppedForVirtCustomize := false
		isRemote := client.IsRemoteConnection()
		var strategyUsed string
		var resetErr error

		// 策略 1: qemu-guest-agent
		if wasRunning {
			logger.Info().
				Str("instance_id", reqCopy.InstanceID).
				Msg("Trying qemu-guest-agent strategy")

			guestAgentStrategy := NewQemuGuestAgentStrategy(client)
			resetErr = guestAgentStrategy.ResetPassword(ctxCopy, reqCopy.InstanceID, usersMap)
			if resetErr == nil {
				strategyUsed = guestAgentStrategy.Name()
				logger.Info().
					Str("instance_id", reqCopy.InstanceID).
					Str("strategy", strategyUsed).
					Msg("Password reset successful via qemu-guest-agent")
			} else {
				logger.Warn().
					Err(resetErr).
					Str("instance_id", reqCopy.InstanceID).
					Msg("qemu-guest-agent strategy failed, trying cloud-init")
			}
		}

		// 策略 2: cloud-init
		if resetErr != nil && wasRunning {
			logger.Info().
				Str("instance_id", reqCopy.InstanceID).
				Msg("Trying cloud-init strategy")

			cloudInitStrategy := NewCloudInitStrategy(client, "")
			resetErr = cloudInitStrategy.ResetPassword(ctxCopy, reqCopy.InstanceID, usersMap)
			if resetErr == nil {
				strategyUsed = cloudInitStrategy.Name()
				logger.Info().
					Str("instance_id", reqCopy.InstanceID).
					Str("strategy", strategyUsed).
					Msg("Password reset successful via cloud-init (requires restart)")
			} else {
				logger.Warn().
					Err(resetErr).
					Str("instance_id", reqCopy.InstanceID).
					Msg("cloud-init strategy failed, falling back to virt-customize")
			}
		}

		// 策略 3: virt-customize
		if resetErr != nil || !wasRunning {
			logger.Info().
				Str("instance_id", reqCopy.InstanceID).
				Msg("Trying virt-customize strategy")

			if s.virtCustomizeClient == nil && !isRemote {
				logger.Error().
					Str("instance_id", reqCopy.InstanceID).
					Msg("virt-customize command not found")
				return
			}

			// 如果实例正在运行，需要先停止
			if wasRunning {
				logger.Info().
					Str("instance_id", reqCopy.InstanceID).
					Msg("Stopping instance before virt-customize password reset")

				stopReq := &entity.StopInstancesRequest{
					NodeName:    reqCopy.NodeName,
					InstanceIDs: []string{reqCopy.InstanceID},
					Force:       false,
				}
				if _, err := s.StopInstances(ctxCopy, stopReq); err != nil {
					logger.Error().
						Err(err).
						Str("instance_id", reqCopy.InstanceID).
						Msg("Failed to stop instance before virt-customize")
					return
				}
				stoppedForVirtCustomize = true

				maxWait := 30 * time.Second
				waitInterval := 1 * time.Second
				waited := time.Duration(0)
				for waited < maxWait {
					inst, err := s.GetInstance(ctxCopy, reqCopy.NodeName, reqCopy.InstanceID)
					if err == nil && inst.State == "stopped" {
						break
					}
					time.Sleep(waitInterval)
					waited += waitInterval
				}

				inst, err := s.GetInstance(ctxCopy, reqCopy.NodeName, reqCopy.InstanceID)
				if err != nil || inst.State != "stopped" {
					logger.Error().
						Str("instance_id", reqCopy.InstanceID).
						Msg("Instance failed to stop within timeout for virt-customize")
					return
				}
			}

			disks, err := client.GetDomainDisks(reqCopy.InstanceID)
			if err != nil {
				logger.Error().
					Err(err).
					Str("instance_id", reqCopy.InstanceID).
					Msg("Failed to get instance disks")
				return
			}

			if len(disks) == 0 || disks[0].Source.File == "" {
				logger.Error().
					Str("instance_id", reqCopy.InstanceID).
					Msg("Instance has no disk for virt-customize")
				return
			}

			diskPath := disks[0].Source.File

			if isRemote {
				passwordArgs := make([]string, 0, len(usersMap)*2)
				for user, pwd := range usersMap {
					passwordArgs = append(passwordArgs, "--password", fmt.Sprintf("%s:password:%s", user, pwd))
				}
				checkCmd := fmt.Sprintf("test -f '%s'", diskPath)
				if err := client.ExecuteRemoteCommand(checkCmd); err != nil {
					resetErr = fmt.Errorf("validate remote disk path: %w", err)
				} else {
					cmd := fmt.Sprintf("virt-customize -a '%s' %s", diskPath, strings.Join(passwordArgs, " "))
					resetErr = client.ExecuteRemoteCommand(cmd)
				}
				if resetErr == nil {
					strategyUsed = "virt-customize-remote"
					logger.Info().
						Str("instance_id", reqCopy.InstanceID).
						Str("strategy", strategyUsed).
						Msg("Password reset successful via virt-customize on remote node")
				}
			} else {
				virtCustomizeStrategy := NewVirtCustomizeStrategy(s.virtCustomizeClient, client)
				resetErr = virtCustomizeStrategy.ResetPassword(ctxCopy, diskPath, usersMap)
				if resetErr == nil {
					strategyUsed = virtCustomizeStrategy.Name()
					logger.Info().
						Str("instance_id", reqCopy.InstanceID).
						Str("strategy", strategyUsed).
						Msg("Password reset successful via virt-customize")
				}
			}
		}

		if resetErr != nil {
			logger.Error().
				Err(resetErr).
				Str("instance_id", reqCopy.InstanceID).
				Msg("Password reset failed")

			if wasRunning && reqCopy.AutoStart && stoppedForVirtCustomize {
				_, _ = s.StartInstances(ctxCopy, &entity.StartInstancesRequest{
					NodeName:    reqCopy.NodeName,
					InstanceIDs: []string{reqCopy.InstanceID},
				})
			}
			return
		}

		if wasRunning && reqCopy.AutoStart && strings.HasPrefix(strategyUsed, "virt-customize") && stoppedForVirtCustomize {
			logger.Info().
				Str("instance_id", reqCopy.InstanceID).
				Msg("Starting instance after password reset")

			if _, err := s.StartInstances(ctxCopy, &entity.StartInstancesRequest{
				NodeName:    reqCopy.NodeName,
				InstanceIDs: []string{reqCopy.InstanceID},
			}); err != nil {
				logger.Warn().
					Err(err).
					Str("instance_id", reqCopy.InstanceID).
					Msg("Failed to start instance after password reset")
			}
		}

		if strategyUsed == "" {
			strategyUsed = "qemu-guest-agent"
		}

		logger.Info().
			Str("instance_id", reqCopy.InstanceID).
			Str("strategy", strategyUsed).
			Strs("users", userList).
			Msg("Password reset completed")
	})

	// 立即返回接受状态
	return &entity.ResetPasswordResponse{
		InstanceID: req.InstanceID,
		Success:    true,
		Message:    "Password reset task started asynchronously",
		Users:      userList,
	}, nil
}

// ListVMTemplates 列出所有可用的 VM 模板
// VM Template 是指带有快照的虚拟机，可以基于快照克隆新的 VM
func (s *InstanceService) ListVMTemplates(ctx context.Context, nodeName string) ([]entity.VMTemplate, error) {
	logger := zerolog.Ctx(ctx)

	// 获取节点的 libvirt 客户端
	client, err := s.nodeProvider.GetNodeStorage(ctx, nodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 直接从 libvirt 获取所有 domain（包括不在 metadata store 中的）
	domains, err := client.GetVMSummaries()
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs from libvirt: %w", err)
	}

	templates := make([]entity.VMTemplate, 0)

	// 遍历每个 domain，检查是否有快照
	for _, domain := range domains {
		// 获取 domain 的快照列表
		snapshots, err := client.ListSnapshots(domain.Name)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to list snapshots for domain")
			continue
		}

		logger.Debug().
			Str("domain_name", domain.Name).
			Int("snapshot_count", len(snapshots)).
			Msg("Checked domain for snapshots")

		// 如果 domain 有快照，将其作为模板
		if len(snapshots) > 0 {
			// 获取 domain 详细信息
			domainInfo, err := client.GetDomainInfo(domain.UUID)
			if err != nil {
				logger.Warn().
					Str("domain_name", domain.Name).
					Err(err).
					Msg("Failed to get domain info")
				continue
			}

			template := entity.VMTemplate{
				ID:          formatDomainUUID(domain.UUID),
				Name:        domain.Name + "-template",
				Description: fmt.Sprintf("Template based on %s with %d snapshots", domain.Name, len(snapshots)),
				SourceVM:    domain.Name,
				VCPUs:       domainInfo.VCPUs,
				Memory:      domainInfo.Memory / 1024, // 转换为 MB
				DiskSize:    20,                       // 默认磁盘大小，可以后续优化从实际磁盘获取
				CreatedAt:   time.Now().Format(time.RFC3339),
			}
			templates = append(templates, template)

			logger.Debug().
				Str("domain_name", domain.Name).
				Str("template_id", template.ID).
				Msg("Added domain as template")
		}
	}

	logger.Info().
		Int("template_count", len(templates)).
		Msg("Listed VM templates")

	return templates, nil
}

// sortInstancesByName 按 name 升序排序实例
func (s *InstanceService) sortInstancesByName(instances []entity.Instance) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})
}
