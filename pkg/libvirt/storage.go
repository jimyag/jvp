package libvirt

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/rs/zerolog/log"
)

// StoragePoolInfo 存储池信息
type StoragePoolInfo struct {
	Name        string
	State       string
	CapacityB   uint64
	AllocationB uint64
	AvailableB  uint64
	Path        string
}

// VolumeInfo 存储卷信息
type VolumeInfo struct {
	Name        string
	Path        string
	CapacityB   uint64
	AllocationB uint64
	Format      string
}

// StoragePoolXML 存储池 XML 结构
// Reference: https://libvirt.org/formatstorage.html
type StoragePoolXML struct {
	XMLName xml.Name   `xml:"pool"`
	Type    string     `xml:"type,attr"`
	Name    string     `xml:"name"`
	Target  PoolTarget `xml:"target"`
}

// PoolTarget 存储池目标配置
type PoolTarget struct {
	Path string `xml:"path"`
}

// VolumeXML 存储卷 XML 结构
// Reference: https://libvirt.org/formatstorage.html#StorageVol
type VolumeXML struct {
	XMLName      xml.Name            `xml:"volume"`
	Type         string              `xml:"type,attr,omitempty"`
	Name         string              `xml:"name"`
	Capacity     VolumeSize          `xml:"capacity"`
	Allocation   VolumeSize          `xml:"allocation"`
	Target       VolumeTarget        `xml:"target"`
	BackingStore *VolumeBackingStore `xml:"backingStore,omitempty"`
}

// VolumeBackingStore 存储卷 backing store 配置（用于创建增量卷）
type VolumeBackingStore struct {
	Path   string       `xml:"path"`
	Format VolumeFormat `xml:"format"`
}

// VolumeSize 存储卷大小配置
type VolumeSize struct {
	Unit  string `xml:"unit,attr"`
	Value uint64 `xml:",chardata"`
}

// VolumeTarget 存储卷目标配置
type VolumeTarget struct {
	Format VolumeFormat `xml:"format"`
}

// VolumeFormat 存储卷格式配置
type VolumeFormat struct {
	Type string `xml:"type,attr"`
}

// mapStoragePoolState 将 libvirt 的 pool 状态转换为字符串
func mapStoragePoolState(s uint8) string {
	switch libvirt.StoragePoolState(s) {
	case libvirt.StoragePoolInactive:
		return "Inactive"
	case libvirt.StoragePoolBuilding:
		return "Building"
	case libvirt.StoragePoolRunning:
		return "Active"
	case libvirt.StoragePoolDegraded:
		return "Degraded"
	case libvirt.StoragePoolInaccessible:
		return "Inaccessible"
	default:
		return "Unknown"
	}
}

// GetStoragePool 获取存储池信息
func (c *Client) GetStoragePool(poolName string) (*StoragePoolInfo, error) {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	state, capacity, allocation, available, err := c.conn.StoragePoolGetInfo(pool)
	if err != nil {
		return nil, fmt.Errorf("get pool info: %w", err)
	}

	// 获取 pool 路径
	xmlDesc, err := c.conn.StoragePoolGetXMLDesc(pool, 0)
	if err != nil {
		return nil, fmt.Errorf("get pool XML: %w", err)
	}

	path := extractPoolPath(xmlDesc)

	return &StoragePoolInfo{
		Name:        poolName,
		State:       mapStoragePoolState(state),
		CapacityB:   capacity,
		AllocationB: allocation,
		AvailableB:  available,
		Path:        path,
	}, nil
}

// ListStoragePools 列出所有存储池
func (c *Client) ListStoragePools() ([]*StoragePoolInfo, error) {
	// 使用新的 API ConnectListAllStoragePools 替代已弃用的 StoragePools
	// NeedResults: 设置为足够大的数字以获取所有 pools
	// Flags: 0 表示获取所有类型的 pools
	pools, _, err := c.conn.ConnectListAllStoragePools(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("list storage pools: %w", err)
	}

	result := make([]*StoragePoolInfo, 0, len(pools))
	for _, p := range pools {
		// 获取 pool 名称（需要通过 XML 或其他方式）
		xmlDesc, err := c.conn.StoragePoolGetXMLDesc(p, 0)
		if err != nil {
			continue
		}

		name := extractPoolName(xmlDesc)
		state, capacity, allocation, available, err := c.conn.StoragePoolGetInfo(p)
		if err != nil {
			continue
		}

		path := extractPoolPath(xmlDesc)

		result = append(result, &StoragePoolInfo{
			Name:        name,
			State:       mapStoragePoolState(state),
			CapacityB:   capacity,
			AllocationB: allocation,
			AvailableB:  available,
			Path:        path,
		})
	}

	return result, nil
}

// EnsureStoragePool 确保存储池存在，如果不存在则创建
func (c *Client) EnsureStoragePool(poolName, poolType, poolPath string) error {
	// 先尝试查找 pool
	_, err := c.conn.StoragePoolLookupByName(poolName)
	if err == nil {
		// Pool 已存在
		return nil
	}

	// Pool 不存在，创建它
	return c.CreateStoragePool(poolName, poolType, poolPath)
}

// CreateStoragePool 创建存储池
func (c *Client) CreateStoragePool(poolName, poolType, poolPath string) error {
	if poolType == "" {
		poolType = "dir" // 默认类型
	}

	// 注意：不要在这里预先创建目录！
	// libvirt 的 StoragePoolBuild 会自动创建目录，并使用正确的权限
	// 无论是本地还是远程节点，libvirt daemon 都会以正确的用户（通常是 root）创建目录

	// 构建 XML 结构
	poolXML := &StoragePoolXML{
		Type: poolType,
		Name: poolName,
		Target: PoolTarget{
			Path: poolPath,
		},
	}

	// 序列化为 XML
	xmlBytes, err := xml.MarshalIndent(poolXML, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pool XML: %w", err)
	}
	xmlDesc := string(xmlBytes)

	// 定义 pool
	pool, err := c.conn.StoragePoolDefineXML(xmlDesc, 0)
	if err != nil {
		return fmt.Errorf("define storage pool: %w", err)
	}

	// 构建 pool（创建目录结构）
	if err := c.conn.StoragePoolBuild(pool, libvirt.StoragePoolBuildNew); err != nil {
		return fmt.Errorf("build storage pool: %w", err)
	}

	// 启动 pool
	if err := c.conn.StoragePoolCreate(pool, libvirt.StoragePoolCreateNormal); err != nil {
		return fmt.Errorf("start storage pool: %w", err)
	}

	// 设置自动启动
	if err := c.conn.StoragePoolSetAutostart(pool, 1); err != nil {
		// 非致命错误，只记录
		log.Warn().Err(err).Str("pool", poolName).Msg("Failed to set pool autostart")
	}

	return nil
}

// GetVolume 获取存储卷信息
func (c *Client) GetVolume(poolName, volumeName string) (*VolumeInfo, error) {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	vol, err := c.conn.StorageVolLookupByName(pool, volumeName)
	if err != nil {
		return nil, fmt.Errorf("lookup volume %s: %w", volumeName, err)
	}

	path, err := c.conn.StorageVolGetPath(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume path: %w", err)
	}

	_, capacity, allocation, err := c.conn.StorageVolGetInfo(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume info: %w", err)
	}

	// 获取格式
	xmlDesc, err := c.conn.StorageVolGetXMLDesc(vol, 0)
	if err != nil {
		return nil, fmt.Errorf("get volume XML: %w", err)
	}
	format := extractVolumeFormat(xmlDesc)

	return &VolumeInfo{
		Name:        volumeName,
		Path:        path,
		CapacityB:   capacity,
		AllocationB: allocation,
		Format:      format,
	}, nil
}

// ListVolumes 列出存储池中的所有卷
func (c *Client) ListVolumes(poolName string) ([]*VolumeInfo, error) {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	// NeedResults: 设置为足够大的数字以获取所有 volumes
	// Flags: 0 表示获取所有类型的 volumes
	vols, _, err := c.conn.StoragePoolListAllVolumes(pool, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	result := make([]*VolumeInfo, 0, len(vols))
	for _, v := range vols {
		path, err := c.conn.StorageVolGetPath(v)
		if err != nil {
			continue
		}

		volType, capacity, allocation, err := c.conn.StorageVolGetInfo(v)
		if err != nil {
			continue
		}

		xmlDesc, err := c.conn.StorageVolGetXMLDesc(v, 0)
		if err != nil {
			continue
		}
		format := extractVolumeFormat(xmlDesc)
		name := extractVolumeName(xmlDesc)

		result = append(result, &VolumeInfo{
			Name:        name,
			Path:        path,
			CapacityB:   capacity,
			AllocationB: allocation,
			Format:      format,
		})
		_ = volType // 暂时不使用
	}

	return result, nil
}

// CreateVolume 创建存储卷
func (c *Client) CreateVolume(poolName, volumeName string, sizeGB uint64, format string) (*VolumeInfo, error) {
	if format == "" {
		format = "qcow2" // 默认格式
	}

	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	// 构建 volume XML 结构
	volumeXML := &VolumeXML{
		Type: "file",
		Name: volumeName,
		Capacity: VolumeSize{
			Unit:  "G",
			Value: sizeGB,
		},
		Allocation: VolumeSize{
			Unit:  "G",
			Value: 0,
		},
		Target: VolumeTarget{
			Format: VolumeFormat{
				Type: format,
			},
		},
	}

	// 序列化为 XML
	xmlBytes, err := xml.MarshalIndent(volumeXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal volume XML: %w", err)
	}
	volXML := string(xmlBytes)

	vol, err := c.conn.StorageVolCreateXML(pool, volXML, 0)
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	// 获取 volume 信息
	path, err := c.conn.StorageVolGetPath(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume path: %w", err)
	}

	volType, capacity, allocation, err := c.conn.StorageVolGetInfo(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume info: %w", err)
	}

	// 修复权限（如果以 root 创建）
	if err := fixVolumeOwnership(c, vol, pool); err != nil {
		// 非致命错误，只记录
		log.Warn().Err(err).Str("volume", volumeName).Msg("Failed to fix volume ownership")
	}

	_ = volType // 暂时不使用

	return &VolumeInfo{
		Name:        volumeName,
		Path:        path,
		CapacityB:   capacity,
		AllocationB: allocation,
		Format:      format,
	}, nil
}

// CreateVolumeWithBackingStore 创建带 backing store 的存储卷（增量卷）
// 用于从模板创建 VM 磁盘，实现 copy-on-write
func (c *Client) CreateVolumeWithBackingStore(poolName, volumeName string, capacityGB uint64, format string, backingPath string, backingFormat string) (*VolumeInfo, error) {
	if format == "" {
		format = "qcow2"
	}
	if backingFormat == "" {
		backingFormat = "qcow2"
	}

	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	// 构建带 backing store 的 volume XML
	volumeXML := &VolumeXML{
		Name: volumeName,
		Capacity: VolumeSize{
			Unit:  "G",
			Value: capacityGB,
		},
		Allocation: VolumeSize{
			Unit:  "G",
			Value: 0, // 增量卷初始分配为 0
		},
		Target: VolumeTarget{
			Format: VolumeFormat{
				Type: format,
			},
		},
		BackingStore: &VolumeBackingStore{
			Path: backingPath,
			Format: VolumeFormat{
				Type: backingFormat,
			},
		},
	}

	// 序列化为 XML
	xmlBytes, err := xml.MarshalIndent(volumeXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal volume XML: %w", err)
	}
	volXML := string(xmlBytes)

	vol, err := c.conn.StorageVolCreateXML(pool, volXML, 0)
	if err != nil {
		return nil, fmt.Errorf("create volume with backing store: %w", err)
	}

	// 获取 volume 信息
	path, err := c.conn.StorageVolGetPath(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume path: %w", err)
	}

	volType, capacity, allocation, err := c.conn.StorageVolGetInfo(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume info: %w", err)
	}

	// 修复权限
	if err := fixVolumeOwnership(c, vol, pool); err != nil {
		log.Warn().Err(err).Str("volume", volumeName).Msg("Failed to fix volume ownership")
	}

	_ = volType

	return &VolumeInfo{
		Name:        volumeName,
		Path:        path,
		CapacityB:   capacity,
		AllocationB: allocation,
		Format:      format,
	}, nil
}

// UploadFileToPool 上传本地文件到存储池
// 用于上传 cloud-init ISO 等文件到远程节点
func (c *Client) UploadFileToPool(poolName string, volumeName string, localFilePath string) (*VolumeInfo, error) {
	// 获取本地文件信息
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("stat local file: %w", err)
	}
	fileSize := uint64(fileInfo.Size())

	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	// 先尝试删除已存在的卷（如果存在）
	existingVol, err := c.conn.StorageVolLookupByName(pool, volumeName)
	if err == nil {
		_ = c.conn.StorageVolDelete(existingVol, libvirt.StorageVolDeleteNormal)
	}

	// 创建 raw 格式的卷
	volumeXML := fmt.Sprintf(`<volume type='file'>
  <name>%s</name>
  <capacity unit='B'>%d</capacity>
  <target>
    <format type='raw'/>
  </target>
</volume>`, volumeName, fileSize)

	vol, err := c.conn.StorageVolCreateXML(pool, volumeXML, 0)
	if err != nil {
		return nil, fmt.Errorf("create volume for upload: %w", err)
	}

	// 打开本地文件
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("open local file: %w", err)
	}
	defer localFile.Close()

	// 读取文件内容
	content, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("read local file: %w", err)
	}

	// 使用 StorageVolUpload 上传
	// 注意：go-libvirt 的 StorageVolUpload 需要通过 stream 传输
	// 由于 go-libvirt 对 stream 的支持有限，我们使用 SSH 方式作为后备
	if c.IsRemoteConnection() {
		// 远程连接使用 SSH 上传
		sshTarget, err := c.GetSSHTarget()
		if err != nil {
			return nil, fmt.Errorf("get SSH target: %w", err)
		}

		// 获取卷路径
		volPath, err := c.conn.StorageVolGetPath(vol)
		if err != nil {
			return nil, fmt.Errorf("get volume path: %w", err)
		}

		// 使用 ssh + cat 上传文件内容
		cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget,
			fmt.Sprintf("cat > '%s'", volPath))
		cmd.Stdin = localFile
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("ssh upload failed: %w, output: %s", err, string(output))
		}
	} else {
		// 本地连接直接写入
		volPath, err := c.conn.StorageVolGetPath(vol)
		if err != nil {
			return nil, fmt.Errorf("get volume path: %w", err)
		}
		if err := os.WriteFile(volPath, content, 0o644); err != nil {
			return nil, fmt.Errorf("write to volume: %w", err)
		}
	}

	// 刷新存储池以更新卷信息
	if err := c.conn.StoragePoolRefresh(pool, 0); err != nil {
		// 非致命错误
		log.Warn().Err(err).Str("pool", poolName).Msg("Failed to refresh pool")
	}

	// 获取卷信息
	path, err := c.conn.StorageVolGetPath(vol)
	if err != nil {
		return nil, fmt.Errorf("get volume path: %w", err)
	}

	return &VolumeInfo{
		Name:        volumeName,
		Path:        path,
		CapacityB:   fileSize,
		AllocationB: fileSize,
		Format:      "raw",
	}, nil
}

// DeleteVolume 删除存储卷
func (c *Client) DeleteVolume(poolName, volumeName string) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	vol, err := c.conn.StorageVolLookupByName(pool, volumeName)
	if err != nil {
		return fmt.Errorf("lookup volume %s: %w", volumeName, err)
	}

	if err := c.conn.StorageVolDelete(vol, libvirt.StorageVolDeleteNormal); err != nil {
		return fmt.Errorf("delete volume: %w", err)
	}

	return nil
}

// ResizeVolume 调整存储卷大小
func (c *Client) ResizeVolume(poolName, volumeName string, newSizeGB uint64) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	vol, err := c.conn.StorageVolLookupByName(pool, volumeName)
	if err != nil {
		return fmt.Errorf("lookup volume %s: %w", volumeName, err)
	}

	// 转换为字节
	newCapacity := newSizeGB * 1024 * 1024 * 1024

	// 使用 libvirt API 调整大小
	if err := c.conn.StorageVolResize(vol, newCapacity, 0); err != nil {
		return fmt.Errorf("resize volume: %w", err)
	}

	return nil
}

// extractPoolPath 从 pool XML 中提取路径
func extractPoolPath(xmlDesc string) string {
	// 优先查找 <target><path> 标签（这是存储池的实际路径）
	targetStart := strings.Index(xmlDesc, "<target>")
	if targetStart != -1 {
		targetEnd := strings.Index(xmlDesc[targetStart:], "</target>")
		if targetEnd != -1 {
			targetSection := xmlDesc[targetStart : targetStart+targetEnd]
			pathStart := strings.Index(targetSection, "<path>")
			if pathStart != -1 {
				pathStart += len("<path>")
				pathEnd := strings.Index(targetSection[pathStart:], "</path>")
				if pathEnd != -1 {
					return targetSection[pathStart : pathStart+pathEnd]
				}
			}
		}
	}

	// 回退到查找任意 <path> 标签
	pathStart := strings.Index(xmlDesc, "<path>")
	if pathStart == -1 {
		return ""
	}
	pathStart += len("<path>")
	pathEnd := strings.Index(xmlDesc[pathStart:], "</path>")
	if pathEnd == -1 {
		return ""
	}
	return xmlDesc[pathStart : pathStart+pathEnd]
}

// extractPoolName 从 pool XML 中提取名称
func extractPoolName(xmlDesc string) string {
	// 查找 <name> 标签
	nameStart := strings.Index(xmlDesc, "<name>")
	if nameStart == -1 {
		return ""
	}
	nameStart += len("<name>")
	nameEnd := strings.Index(xmlDesc[nameStart:], "</name>")
	if nameEnd == -1 {
		return ""
	}
	return xmlDesc[nameStart : nameStart+nameEnd]
}

// extractVolumeName 从 volume XML 中提取名称
func extractVolumeName(xmlDesc string) string {
	// 查找 <name> 标签
	nameStart := strings.Index(xmlDesc, "<name>")
	if nameStart == -1 {
		return ""
	}
	nameStart += len("<name>")
	nameEnd := strings.Index(xmlDesc[nameStart:], "</name>")
	if nameEnd == -1 {
		return ""
	}
	return xmlDesc[nameStart : nameStart+nameEnd]
}

// extractVolumeFormat 从 volume XML 中提取格式
func extractVolumeFormat(xmlDesc string) string {
	// 查找 format type='...'
	formatStart := strings.Index(xmlDesc, "format type='")
	if formatStart == -1 {
		return "unknown"
	}
	formatStart += len("format type='")
	formatEnd := strings.Index(xmlDesc[formatStart:], "'")
	if formatEnd == -1 {
		return "unknown"
	}
	return xmlDesc[formatStart : formatStart+formatEnd]
}

// fixVolumeOwnership 修复 volume 的所有权（从 pool 继承）
func fixVolumeOwnership(c *Client, vol libvirt.StorageVol, pool libvirt.StoragePool) error {
	volPath, err := c.conn.StorageVolGetPath(vol)
	if err != nil {
		return err
	}

	// 检查文件是否存在
	info, err := os.Stat(volPath)
	if err != nil {
		return err
	}

	// 如果文件是 root 拥有的，尝试修复
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid == 0 {
		// 从 pool XML 获取 owner/group
		poolXML, err := c.conn.StoragePoolGetXMLDesc(pool, 0)
		if err != nil {
			return err
		}

		// 提取 owner 和 group
		ownerID, groupID, err := extractOwnerGroupFromXML(poolXML)
		if err != nil {
			return err
		}

		if ownerID > 0 && groupID > 0 {
			return os.Chown(volPath, int(ownerID), int(groupID))
		}
	}

	return nil
}

// extractOwnerGroupFromXML 从 XML 中提取 owner 和 group
func extractOwnerGroupFromXML(xmlDesc string) (int, int, error) {
	var ownerID, groupID uint64
	var err error

	// 提取 owner
	if strings.Contains(xmlDesc, "<owner>") {
		ownerStart := strings.Index(xmlDesc, "<owner>") + 7
		ownerEnd := strings.Index(xmlDesc[ownerStart:], "</owner>")
		if ownerEnd > 0 {
			ownerStr := xmlDesc[ownerStart : ownerStart+ownerEnd]
			ownerID, err = strconv.ParseUint(ownerStr, 10, 32)
			if err != nil {
				return 0, 0, err
			}
		}
	}

	// 提取 group
	if strings.Contains(xmlDesc, "<group>") {
		groupStart := strings.Index(xmlDesc, "<group>") + 7
		groupEnd := strings.Index(xmlDesc[groupStart:], "</group>")
		if groupEnd > 0 {
			groupStr := xmlDesc[groupStart : groupStart+groupEnd]
			groupID, err = strconv.ParseUint(groupStr, 10, 32)
			if err != nil {
				return 0, 0, err
			}
		}
	}

	return int(ownerID), int(groupID), nil
}

// StartStoragePool 启动存储池
func (c *Client) StartStoragePool(poolName string) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	if err := c.conn.StoragePoolCreate(pool, 0); err != nil {
		return fmt.Errorf("start storage pool %s: %w", poolName, err)
	}

	return nil
}

// StopStoragePool 停止存储池
func (c *Client) StopStoragePool(poolName string) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	if err := c.conn.StoragePoolDestroy(pool); err != nil {
		return fmt.Errorf("stop storage pool %s: %w", poolName, err)
	}

	return nil
}

// DeleteStoragePool 删除存储池
// deleteVolumes: 是否同时删除存储池中的所有卷和目录
func (c *Client) DeleteStoragePool(poolName string, deleteVolumes bool) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	// 如果需要删除卷和目录
	if deleteVolumes {
		// 先刷新 pool 以确保卷列表是最新的
		_ = c.conn.StoragePoolRefresh(pool, 0)

		// 获取所有卷
		vols, _, err := c.conn.StoragePoolListAllVolumes(pool, 1000, 0)
		if err == nil {
			// 删除所有卷
			for _, vol := range vols {
				_ = c.conn.StorageVolDelete(vol, libvirt.StorageVolDeleteNormal)
			}
		}

		// 先停止 pool（如果正在运行）
		_ = c.conn.StoragePoolDestroy(pool)

		// 删除 pool（包括目录）
		if err := c.conn.StoragePoolDelete(pool, libvirt.StoragePoolDeleteNormal); err != nil {
			// 如果删除失败，可能是因为目录不为空或没有权限，只记录错误
			log.Warn().Err(err).Str("pool", poolName).Msg("Failed to delete pool directory")
		}
	} else {
		// 只停止 pool，不删除目录
		_ = c.conn.StoragePoolDestroy(pool)
	}

	// 取消定义 pool
	if err := c.conn.StoragePoolUndefine(pool); err != nil {
		return fmt.Errorf("undefine storage pool %s: %w", poolName, err)
	}

	return nil
}

// RefreshStoragePool 刷新存储池
func (c *Client) RefreshStoragePool(poolName string) error {
	pool, err := c.conn.StoragePoolLookupByName(poolName)
	if err != nil {
		return fmt.Errorf("lookup storage pool %s: %w", poolName, err)
	}

	if err := c.conn.StoragePoolRefresh(pool, 0); err != nil {
		return fmt.Errorf("refresh storage pool %s: %w", poolName, err)
	}

	return nil
}

// GetConnectionURI 获取当前连接的 URI
// 返回创建连接时使用的原始 URI（而非远程 libvirtd 返回的 URI）
func (c *Client) GetConnectionURI() string {
	return c.uri
}

// IsRemoteConnection 判断是否是远程连接
func (c *Client) IsRemoteConnection() bool {
	uri := c.GetConnectionURI()
	if uri == "" {
		return false
	}

	// 解析 URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// 检查是否是远程连接（包含 ssh、tcp 等）
	scheme := strings.ToLower(parsedURI.Scheme)
	return strings.Contains(scheme, "ssh") ||
		strings.Contains(scheme, "tcp") ||
		strings.Contains(scheme, "tls") ||
		parsedURI.Host != ""
}

// GetSSHTarget 从 libvirt URI 获取 SSH 目标
// 返回格式: user@host
func (c *Client) GetSSHTarget() (string, error) {
	uri := c.GetConnectionURI()
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse URI: %w", err)
	}

	host := parsedURI.Hostname()
	user := parsedURI.User.Username()
	if user == "" {
		user = "root"
	}

	return fmt.Sprintf("%s@%s", user, host), nil
}

// ExecuteRemoteCommand 在远程节点执行命令
func (c *Client) ExecuteRemoteCommand(command string) error {
	if !c.IsRemoteConnection() {
		return fmt.Errorf("not a remote connection")
	}

	sshTarget, err := c.GetSSHTarget()
	if err != nil {
		return err
	}

	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// ReadRemoteFile 从远程节点读取文件
func (c *Client) ReadRemoteFile(path string) ([]byte, error) {
	if !c.IsRemoteConnection() {
		return nil, fmt.Errorf("not a remote connection")
	}

	sshTarget, err := c.GetSSHTarget()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, fmt.Sprintf("cat '%s'", path))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ssh cat failed: %w", err)
	}

	return output, nil
}

// ListRemoteFiles 列举远程目录中的文件
func (c *Client) ListRemoteFiles(dir, pattern string) ([]string, error) {
	if !c.IsRemoteConnection() {
		return nil, fmt.Errorf("not a remote connection")
	}

	sshTarget, err := c.GetSSHTarget()
	if err != nil {
		return nil, err
	}

	// 使用 find 命令查找匹配的文件
	findCmd := fmt.Sprintf("find '%s' -maxdepth 1 -name '%s' -type f -printf '%%f\\n' 2>/dev/null || ls '%s'/%s 2>/dev/null | xargs -n1 basename 2>/dev/null", dir, pattern, dir, pattern)
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, findCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ssh find failed: %w", err)
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// CreateCloudInitISO 在远程（或本地）节点创建 cloud-init ISO
// 返回生成的 ISO 文件路径
func (c *Client) CreateCloudInitISO(outputDir, vmName, metaData, userData string) (string, error) {
	isoPath := fmt.Sprintf("%s/%s-cidata.iso", outputDir, vmName)

	if c.IsRemoteConnection() {
		return c.createCloudInitISORemote(outputDir, vmName, metaData, userData, isoPath)
	}
	return c.createCloudInitISOLocal(outputDir, vmName, metaData, userData, isoPath)
}

// createCloudInitISOLocal 在本地创建 cloud-init ISO
func (c *Client) createCloudInitISOLocal(outputDir, vmName, metaData, userData, isoPath string) (string, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "cloudinit-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 写入 meta-data
	metaDataPath := tmpDir + "/meta-data"
	if err := os.WriteFile(metaDataPath, []byte(metaData), 0o644); err != nil {
		return "", fmt.Errorf("write meta-data: %w", err)
	}

	// 写入 user-data
	userDataPath := tmpDir + "/user-data"
	if err := os.WriteFile(userDataPath, []byte(userData), 0o644); err != nil {
		return "", fmt.Errorf("write user-data: %w", err)
	}

	// 生成 ISO
	var cmd *exec.Cmd
	if _, err := exec.LookPath("genisoimage"); err == nil {
		cmd = exec.Command("genisoimage", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock", tmpDir)
	} else if _, err := exec.LookPath("mkisofs"); err == nil {
		cmd = exec.Command("mkisofs", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock", tmpDir)
	} else {
		return "", fmt.Errorf("neither genisoimage nor mkisofs found")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create ISO: %w, output: %s", err, string(output))
	}

	return isoPath, nil
}

// createCloudInitISORemote 在远程节点创建 cloud-init ISO
func (c *Client) createCloudInitISORemote(outputDir, vmName, metaData, userData, isoPath string) (string, error) {
	sshTarget, err := c.GetSSHTarget()
	if err != nil {
		return "", err
	}

	// 在远程创建临时目录
	tmpDir := fmt.Sprintf("/tmp/cloudinit-%s-%d", vmName, time.Now().UnixNano())
	mkdirCmd := fmt.Sprintf("mkdir -p '%s'", tmpDir)
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, mkdirCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create remote temp dir: %w, output: %s", err, string(output))
	}

	// 清理临时目录
	defer func() {
		cleanCmd := fmt.Sprintf("rm -rf '%s'", tmpDir)
		cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, cleanCmd)
		_ = cmd.Run()
	}()

	// 写入 meta-data 到远程
	metaDataPath := tmpDir + "/meta-data"
	writeMetaCmd := fmt.Sprintf("cat > '%s'", metaDataPath)
	cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, writeMetaCmd)
	cmd.Stdin = strings.NewReader(metaData)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("write remote meta-data: %w, output: %s", err, string(output))
	}

	// 写入 user-data 到远程
	userDataPath := tmpDir + "/user-data"
	writeUserCmd := fmt.Sprintf("cat > '%s'", userDataPath)
	cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, writeUserCmd)
	cmd.Stdin = strings.NewReader(userData)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("write remote user-data: %w, output: %s", err, string(output))
	}

	// 在远程生成 ISO（尝试 genisoimage 或 mkisofs）
	genISOCmd := fmt.Sprintf("(which genisoimage && genisoimage -output '%s' -volid cidata -joliet -rock '%s') || (which mkisofs && mkisofs -output '%s' -volid cidata -joliet -rock '%s')", isoPath, tmpDir, isoPath, tmpDir)
	cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", sshTarget, genISOCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create remote ISO: %w, output: %s", err, string(output))
	}

	return isoPath, nil
}
