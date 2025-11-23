package metadata

import (
	"encoding/xml"
	"sync"
	"time"
)

// JVP 命名空间
const (
	JVPNamespace = "https://github.com/jimyag/jvp/ns/1.0"
	JVPPrefix    = "jvp"
)

// ==================== Domain Metadata 结构 ====================

// JVPInstanceMetadata 存储在 libvirt domain metadata 中的 JVP 元数据
// 匹配 entity.Instance 的简化结构
type JVPInstanceMetadata struct {
	XMLName   xml.Name `xml:"https://github.com/jimyag/jvp/ns/1.0 instance"`
	ID        string   `xml:"id"`
	Name      string   `xml:"name,omitempty"`
	ImageID   string   `xml:"image_id,omitempty"`
	VolumeID  string   `xml:"volume_id,omitempty"`
	CreatedAt string   `xml:"created_at"`
	UpdatedAt string   `xml:"updated_at"`
}

// ==================== 边车文件结构 ====================

// VolumeMetadata 卷的边车元数据文件
type VolumeMetadata struct {
	Version       string            `json:"version"`
	SchemaVersion string            `json:"schema_version"`
	ResourceType  string            `json:"resource_type"`
	ID            string            `json:"id"`
	SnapshotID    string            `json:"snapshot_id,omitempty"`
	VolumeType    string            `json:"volume_type"`
	Iops          int               `json:"iops,omitempty"`
	Encrypted     bool              `json:"encrypted"`
	KmsKeyID      string            `json:"kms_key_id,omitempty"`
	State         string            `json:"state"`
	CreateTime    time.Time         `json:"create_time"`
	UpdateTime    time.Time         `json:"update_time"`
	Tags          map[string]string `json:"tags"`
}

// ImageMetadata 镜像的边车元数据文件
type ImageMetadata struct {
	Version       string            `json:"version"`
	SchemaVersion string            `json:"schema_version"`
	ResourceType  string            `json:"resource_type"`
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Pool          string            `json:"pool"`
	Path          string            `json:"path"`
	Format        string            `json:"format"`
	State         string            `json:"state"`
	CreatedAt     string            `json:"created_at"`
	Tags          map[string]string `json:"tags"`
}

// SnapshotIndex 快照索引文件 (存储在 .snapshots/ 目录下)
type SnapshotIndex struct {
	VolumeID   string     `json:"volume_id"`
	VolumePath string     `json:"volume_path"`
	Snapshots  []Snapshot `json:"snapshots"`
}

// Snapshot 快照元数据
type Snapshot struct {
	ID               string            `json:"id"`
	Name             string            `json:"name,omitempty"`
	Description      string            `json:"description,omitempty"`
	QemuSnapshotName string            `json:"qemu_snapshot_name"`
	SizeGB           uint64            `json:"size_gb"`
	State            string            `json:"state"`
	Progress         int               `json:"progress"`
	StartTime        time.Time         `json:"start_time"`
	CompletionTime   time.Time         `json:"completion_time,omitempty"`
	OwnerID          string            `json:"owner_id,omitempty"`
	Tags             map[string]string `json:"tags"`
}

// KeyPairMetadata 密钥对元数据
type KeyPairMetadata struct {
	Version       string `json:"version"`
	SchemaVersion string `json:"schema_version"`
	ResourceType  string `json:"resource_type"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Algorithm     string `json:"algorithm"`   // rsa, ed25519
	Fingerprint   string `json:"fingerprint"` // SHA256
	PublicKey     string `json:"public_key"`
	CreatedAt     string `json:"created_at"`
}

// KeyPairIndex 密钥对索引文件
type KeyPairIndex struct {
	Version     string             `json:"version"`
	LastUpdated time.Time          `json:"last_updated"`
	KeyPairs    []KeyPairIndexItem `json:"keypairs"`
}

// KeyPairIndexItem 密钥对索引项
type KeyPairIndexItem struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Fingerprint string            `json:"fingerprint"`
	Tags        map[string]string `json:"tags"`
}

// ==================== 内存索引结构 ====================

// MemoryIndex 内存索引,用于加速查询
type MemoryIndex struct {
	sync.RWMutex

	// 实例索引
	Instances        map[string]*InstanceIndex // instance_id -> index
	InstancesByState map[string][]string       // state -> [instance_ids]
	InstancesByImage map[string][]string       // image_id -> [instance_ids]
	InstancesByTag   map[string][]string       // "key=value" -> [instance_ids]

	// 卷索引
	Volumes        map[string]*VolumeIndex // volume_id -> index
	VolumesByType  map[string][]string     // volume_type -> [volume_ids]
	VolumesByState map[string][]string     // state -> [volume_ids]
	VolumesByTag   map[string][]string

	// 镜像索引
	Images      map[string]*ImageIndex // image_id -> index
	ImagesByTag map[string][]string

	// 快照索引
	Snapshots         map[string]*SnapshotIndexItem // snapshot_id -> index
	SnapshotsByVolume map[string][]string           // volume_id -> [snapshot_ids]

	// 密钥对索引
	KeyPairs map[string]*KeyPairIndexEntry // keypair_id -> index

	LastSync time.Time
}

// InstanceIndex 实例索引
type InstanceIndex struct {
	ID         string
	DomainUUID string
	DomainName string
	State      string
	ImageID    string
	VolumeID   string
}

// VolumeIndex 卷索引
type VolumeIndex struct {
	ID         string
	Path       string
	State      string
	VolumeType string
	SizeGB     uint64
	Tags       map[string]string
}

// ImageIndex 镜像索引
type ImageIndex struct {
	ID     string
	Name   string
	Path   string
	State  string
	SizeGB uint64
}

// SnapshotIndex 快照索引
type SnapshotIndexItem struct {
	ID         string
	VolumeID   string
	State      string
	SizeGB     uint64
	CreateTime time.Time
	Tags       map[string]string
}

// KeyPairIndexEntry 密钥对索引
type KeyPairIndexEntry struct {
	ID          string
	Name        string
	Fingerprint string
}

// NewMemoryIndex 创建新的内存索引
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		Instances:         make(map[string]*InstanceIndex),
		InstancesByState:  make(map[string][]string),
		InstancesByImage:  make(map[string][]string),
		InstancesByTag:    make(map[string][]string),
		Volumes:           make(map[string]*VolumeIndex),
		VolumesByType:     make(map[string][]string),
		VolumesByState:    make(map[string][]string),
		VolumesByTag:      make(map[string][]string),
		Images:            make(map[string]*ImageIndex),
		ImagesByTag:       make(map[string][]string),
		Snapshots:         make(map[string]*SnapshotIndexItem),
		SnapshotsByVolume: make(map[string][]string),
		KeyPairs:          make(map[string]*KeyPairIndexEntry),
	}
}

// ==================== 辅助函数 ====================
// (已移除 Tag 相关辅助函数,因为 entity 使用的是简化结构)
