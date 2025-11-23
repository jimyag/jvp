package metadata

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/rs/zerolog/log"
)

// ==================== Instance 元数据管理 ====================

// SaveInstance 保存实例元数据到 libvirt domain metadata
func (s *LibvirtMetadataStore) SaveInstance(ctx context.Context, instance *entity.Instance) error {
	log.Debug().
		Str("instance_id", instance.ID).
		Str("state", instance.State).
		Msg("Saving instance metadata")

	// 1. 查找 domain
	domain, err := s.getDomainByInstanceID(ctx, instance.ID)
	if err != nil {
		// 如果 domain 不存在,说明是新创建的实例,需要等待 domain 创建完成
		// 这里假设 domain 已经通过其他方式创建(比如通过 libvirt XML)
		return fmt.Errorf("domain not found for instance %s: %w", instance.ID, err)
	}

	// 2. 构建 JVP 元数据
	jvpMeta := JVPInstanceMetadata{
		XMLName:   xml.Name{Space: JVPNamespace, Local: "instance"},
		ID:        instance.ID,
		Name:      instance.Name,
		ImageID:   instance.ImageID,
		VolumeID:  instance.VolumeID,
		CreatedAt: instance.CreatedAt,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	// 3. 序列化为 XML
	metaXML, err := xml.Marshal(jvpMeta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// 4. 设置 domain metadata
	err = s.conn.DomainSetMetadata(
		domain,
		int32(libvirt.DomainMetadataElement),
		libvirt.OptString{string(metaXML)},
		libvirt.OptString{JVPPrefix},
		libvirt.OptString{JVPNamespace},
		libvirt.DomainAffectConfig|libvirt.DomainAffectLive,
	)
	if err != nil {
		return fmt.Errorf("set domain metadata: %w", err)
	}

	// 5. 更新内存索引
	s.updateInstanceIndex(instance)

	log.Info().
		Str("instance_id", instance.ID).
		Str("domain_name", instance.ID).
		Msg("Instance metadata saved successfully")

	return nil
}

// GetInstance 获取单个实例
func (s *LibvirtMetadataStore) GetInstance(ctx context.Context, instanceID string) (*entity.Instance, error) {
	log.Debug().Str("instance_id", instanceID).Msg("Getting instance")

	// 1. 从索引获取 domain UUID
	domain, err := s.getDomainByInstanceID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// 2. 读取实例信息
	instance, err := s.buildInstanceFromDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("build instance from domain: %w", err)
	}

	return instance, nil
}

// ListInstances 列出所有实例
func (s *LibvirtMetadataStore) ListInstances(ctx context.Context) ([]*entity.Instance, error) {
	log.Debug().Msg("Listing all instances")

	// 1. 获取所有 domains
	domains, _, err := s.conn.ConnectListAllDomains(1, 0)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}

	instances := make([]*entity.Instance, 0, len(domains))

	// 2. 遍历 domains,提取 JVP 实例
	for _, domain := range domains {
		// 检查是否有 JVP 元数据
		metaXML, err := s.conn.DomainGetMetadata(
			domain,
			int32(libvirt.DomainMetadataElement),
			libvirt.OptString{JVPNamespace},
			libvirt.DomainAffectConfig,
		)
		if err != nil {
			// 没有 JVP 元数据,跳过
			continue
		}

		if metaXML == "" {
			continue
		}

		// 解析为实例
		instance, err := s.buildInstanceFromDomain(ctx, domain)
		if err != nil {
			log.Warn().
				Str("domain_uuid", uuidToString(domain.UUID)).
				Err(err).
				Msg("Failed to build instance from domain")
			continue
		}

		instances = append(instances, instance)
	}

	log.Debug().Int("count", len(instances)).Msg("Listed instances")
	return instances, nil
}

// DescribeInstances 查询实例(支持过滤)
func (s *LibvirtMetadataStore) DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]*entity.Instance, error) {
	log.Debug().
		Strs("instance_ids", req.InstanceIDs).
		Interface("filters", req.Filters).
		Msg("Describing instances")

	// 如果指定了实例 ID,直接查询
	if len(req.InstanceIDs) > 0 {
		return s.getInstancesByIDs(ctx, req.InstanceIDs)
	}

	// 否则,使用索引进行过滤查询
	candidateIDs := s.filterInstancesByIndex(req)

	// 获取实例详情
	instances := make([]*entity.Instance, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		instance, err := s.GetInstance(ctx, id)
		if err != nil {
			log.Warn().Str("instance_id", id).Err(err).Msg("Failed to get instance")
			continue
		}

		// 应用额外的过滤条件
		if s.matchesFilters(instance, req.Filters) {
			instances = append(instances, instance)
		}
	}

	log.Debug().Int("count", len(instances)).Msg("Described instances")
	return instances, nil
}

// DeleteInstance 删除实例元数据
func (s *LibvirtMetadataStore) DeleteInstance(ctx context.Context, instanceID string) error {
	log.Debug().Str("instance_id", instanceID).Msg("Deleting instance metadata")

	// 1. 查找 domain
	domain, err := s.getDomainByInstanceID(ctx, instanceID)
	if err != nil {
		// Domain 不存在,认为已删除
		return nil
	}

	// 2. 删除 JVP 元数据
	err = s.conn.DomainSetMetadata(
		domain,
		int32(libvirt.DomainMetadataElement),
		libvirt.OptString{""},
		libvirt.OptString{JVPPrefix},
		libvirt.OptString{JVPNamespace},
		libvirt.DomainAffectConfig|libvirt.DomainAffectLive,
	)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to delete domain metadata")
	}

	// 3. 从索引中删除
	s.removeInstanceFromIndex(instanceID)

	log.Info().Str("instance_id", instanceID).Msg("Instance metadata deleted")
	return nil
}

// UpdateInstanceState 更新实例状态
func (s *LibvirtMetadataStore) UpdateInstanceState(ctx context.Context, instanceID string, state string) error {
	log.Debug().
		Str("instance_id", instanceID).
		Str("state", state).
		Msg("Updating instance state")

	// 状态存储在 libvirt domain 的运行时状态中,不需要额外存储
	// 只需要更新索引
	s.index.Lock()
	defer s.index.Unlock()

	if idx, exists := s.index.Instances[instanceID]; exists {
		idx.State = state
	}

	return nil
}

// ==================== 辅助函数 ====================

// buildInstanceFromDomain 从 libvirt domain 构建实例对象
func (s *LibvirtMetadataStore) buildInstanceFromDomain(ctx context.Context, domain libvirt.Domain) (*entity.Instance, error) {
	// 1. 尝试读取 JVP 元数据（可能不存在）
	metaXML, err := s.conn.DomainGetMetadata(
		domain,
		int32(libvirt.DomainMetadataElement),
		libvirt.OptString{JVPNamespace},
		libvirt.DomainAffectConfig,
	)

	var jvpMeta JVPInstanceMetadata
	hasJVPMetadata := err == nil && metaXML != ""
	if hasJVPMetadata {
		if err := xml.Unmarshal([]byte(metaXML), &jvpMeta); err != nil {
			log.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to unmarshal JVP metadata, will use domain name as ID")
			hasJVPMetadata = false
		}
	}

	// 2. 读取 domain 基本信息
	state, _, _, _, _, err := s.conn.DomainGetInfo(domain)
	if err != nil {
		return nil, fmt.Errorf("get domain info: %w", err)
	}

	// 3. 读取 domain XML 获取详细配置
	domainXML, err := s.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, fmt.Errorf("get domain xml: %w", err)
	}

	// 4. 解析 domain XML
	var domainDef struct {
		Name   string `xml:"name"`
		Memory struct {
			Value uint64 `xml:",chardata"`
			Unit  string `xml:"unit,attr"`
		} `xml:"memory"`
		VCPU uint `xml:"vcpu"`
	}
	if err := xml.Unmarshal([]byte(domainXML), &domainDef); err != nil {
		return nil, fmt.Errorf("unmarshal domain xml: %w", err)
	}

	// 5. 映射状态
	stateStr := mapDomainState(state)

	// 6. 构建实例对象
	// 如果没有 JVP 元数据，使用 domain name 作为 ID
	instanceID := domain.Name
	instanceName := domain.Name
	imageID := ""
	volumeID := ""
	createdAt := time.Now().Format(time.RFC3339)

	if hasJVPMetadata {
		instanceID = jvpMeta.ID
		instanceName = jvpMeta.Name
		imageID = jvpMeta.ImageID
		volumeID = jvpMeta.VolumeID
		createdAt = jvpMeta.CreatedAt
	}

	instance := &entity.Instance{
		ID:         instanceID,
		Name:       instanceName,
		State:      stateStr,
		ImageID:    imageID,
		VolumeID:   volumeID,
		MemoryMB:   domainDef.Memory.Value / 1024, // 转换为 MB
		VCPUs:      uint16(domainDef.VCPU),
		CreatedAt:  createdAt,
		DomainUUID: uuidToString(domain.UUID),
		DomainName: domain.Name,
	}

	return instance, nil
}

// mapDomainState 映射 libvirt domain 状态到 JVP 状态
func mapDomainState(state uint8) string {
	switch state {
	case 0: // VIR_DOMAIN_NOSTATE
		return "pending"
	case 1: // VIR_DOMAIN_RUNNING
		return "running"
	case 2: // VIR_DOMAIN_BLOCKED
		return "running"
	case 3: // VIR_DOMAIN_PAUSED
		return "stopped"
	case 4: // VIR_DOMAIN_SHUTDOWN
		return "stopping"
	case 5: // VIR_DOMAIN_SHUTOFF
		return "stopped"
	case 6: // VIR_DOMAIN_CRASHED
		return "terminated"
	case 7: // VIR_DOMAIN_PMSUSPENDED
		return "stopped"
	default:
		return "pending"
	}
}

// getInstancesByIDs 根据实例 ID 列表获取实例
func (s *LibvirtMetadataStore) getInstancesByIDs(ctx context.Context, instanceIDs []string) ([]*entity.Instance, error) {
	instances := make([]*entity.Instance, 0, len(instanceIDs))

	for _, id := range instanceIDs {
		instance, err := s.GetInstance(ctx, id)
		if err != nil {
			log.Warn().Str("instance_id", id).Err(err).Msg("Failed to get instance")
			continue
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// filterInstancesByIndex 使用内存索引过滤实例
func (s *LibvirtMetadataStore) filterInstancesByIndex(req *entity.DescribeInstancesRequest) []string {
	s.index.RLock()
	defer s.index.RUnlock()

	// 如果没有过滤条件,返回所有实例 ID
	if len(req.Filters) == 0 {
		ids := make([]string, 0, len(s.index.Instances))
		for id := range s.index.Instances {
			ids = append(ids, id)
		}
		return ids
	}

	// 使用过滤条件
	candidateSet := make(map[string]bool)
	firstFilter := true

	for _, filter := range req.Filters {
		var matchedIDs []string

		switch filter.Name {
		case "instance-state-name":
			for _, value := range filter.Values {
				if ids, exists := s.index.InstancesByState[value]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		case "image-id":
			for _, value := range filter.Values {
				if ids, exists := s.index.InstancesByImage[value]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		case "tag":
			// tag 格式: "key=value"
			for _, tagPair := range filter.Values {
				if ids, exists := s.index.InstancesByTag[tagPair]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		default:
			// 不支持的过滤器,需要后续在内存中过滤
			continue
		}

		// 合并结果(交集)
		if firstFilter {
			for _, id := range matchedIDs {
				candidateSet[id] = true
			}
			firstFilter = false
		} else {
			// 计算交集
			newSet := make(map[string]bool)
			for _, id := range matchedIDs {
				if candidateSet[id] {
					newSet[id] = true
				}
			}
			candidateSet = newSet
		}
	}

	// 转换为切片
	result := make([]string, 0, len(candidateSet))
	for id := range candidateSet {
		result = append(result, id)
	}

	return result
}

// matchesFilters 检查实例是否匹配过滤条件(用于索引不支持的过滤器)
func (s *LibvirtMetadataStore) matchesFilters(instance *entity.Instance, filters []entity.Filter) bool {
	for _, filter := range filters {
		switch filter.Name {
		case "instance-id":
			matched := false
			for _, value := range filter.Values {
				if instance.ID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		// instance-type 过滤器已移除,因为 entity.Instance 不支持

		// 其他过滤器已经在索引层面处理
		}
	}

	return true
}

// updateInstanceIndex 更新实例索引
func (s *LibvirtMetadataStore) updateInstanceIndex(instance *entity.Instance) {
	s.index.Lock()
	defer s.index.Unlock()

	// 获取 domain UUID (这里需要从 domain 中读取)
	// 简化处理,假设已经在索引中
	idx := &InstanceIndex{
		ID:         instance.ID,
		DomainUUID: instance.DomainUUID,
		DomainName: instance.DomainName,
		State:      instance.State,
		ImageID:    instance.ImageID,
		VolumeID:   instance.VolumeID,
	}

	s.index.Instances[instance.ID] = idx

	// 更新状态索引
	if _, exists := s.index.InstancesByState[instance.State]; !exists {
		s.index.InstancesByState[instance.State] = []string{}
	}
	s.index.InstancesByState[instance.State] = append(
		s.index.InstancesByState[instance.State],
		instance.ID,
	)

	// 更新镜像索引
	if instance.ImageID != "" {
		if _, exists := s.index.InstancesByImage[instance.ImageID]; !exists {
			s.index.InstancesByImage[instance.ImageID] = []string{}
		}
		s.index.InstancesByImage[instance.ImageID] = append(
			s.index.InstancesByImage[instance.ImageID],
			instance.ID,
		)
	}
}

// removeInstanceFromIndex 从索引中删除实例
func (s *LibvirtMetadataStore) removeInstanceFromIndex(instanceID string) {
	s.index.Lock()
	defer s.index.Unlock()

	// 从主索引删除
	idx, exists := s.index.Instances[instanceID]
	if !exists {
		return
	}

	delete(s.index.Instances, instanceID)

	// 从状态索引删除
	if ids, exists := s.index.InstancesByState[idx.State]; exists {
		s.index.InstancesByState[idx.State] = removeFromSlice(ids, instanceID)
	}

	// 从镜像索引删除
	if idx.ImageID != "" {
		if ids, exists := s.index.InstancesByImage[idx.ImageID]; exists {
			s.index.InstancesByImage[idx.ImageID] = removeFromSlice(ids, instanceID)
		}
	}
}

// removeFromSlice 从切片中删除元素
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
