// Package service 提供业务逻辑层的服务实现
package service

import (
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/jinzhu/copier"
)

// instanceEntityToModel 将 entity.Instance 转换为 model.Instance
func instanceEntityToModel(e *entity.Instance) (*model.Instance, error) {
	m := &model.Instance{}
	if err := copier.Copy(m, e); err != nil {
		return nil, err
	}

	// 处理时间字段
	if e.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
			m.CreatedAt = t
		} else {
			m.CreatedAt = time.Now()
		}
	} else {
		m.CreatedAt = time.Now()
	}
	m.UpdatedAt = time.Now()

	return m, nil
}

// instanceModelToEntity 将 model.Instance 转换为 entity.Instance
func instanceModelToEntity(m *model.Instance) (*entity.Instance, error) {
	e := &entity.Instance{}
	if err := copier.Copy(e, m); err != nil {
		return nil, err
	}

	// 处理时间字段
	e.CreatedAt = m.CreatedAt.Format(time.RFC3339)

	return e, nil
}

// imageEntityToModel 将 entity.Image 转换为 model.Image
func imageEntityToModel(e *entity.Image) (*model.Image, error) {
	m := &model.Image{}
	if err := copier.Copy(m, e); err != nil {
		return nil, err
	}

	// 处理时间字段
	if e.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
			m.CreatedAt = t
		} else {
			m.CreatedAt = time.Now()
		}
	} else {
		m.CreatedAt = time.Now()
	}
	m.UpdatedAt = time.Now()

	return m, nil
}

// imageModelToEntity 将 model.Image 转换为 entity.Image
func imageModelToEntity(m *model.Image) (*entity.Image, error) {
	e := &entity.Image{}
	if err := copier.Copy(e, m); err != nil {
		return nil, err
	}

	// 处理时间字段
	e.CreatedAt = m.CreatedAt.Format(time.RFC3339)

	return e, nil
}

// volumeEntityToModel 将 entity.EBSVolume 转换为 model.Volume
// 此函数为未来 VolumeService 集成 repository 时使用
func volumeEntityToModel(e *entity.EBSVolume) (*model.Volume, error) {
	m := &model.Volume{
		ID:               e.VolumeID, // 字段名不同，手动映射
		SizeGB:           e.SizeGB,
		SnapshotID:       e.SnapshotID,
		AvailabilityZone: e.AvailabilityZone,
		State:            e.State,
		VolumeType:       e.VolumeType,
		Iops:             e.Iops,
		Encrypted:        e.Encrypted,
		KmsKeyID:         e.KmsKeyID,
	}

	// 处理时间字段
	if e.CreateTime != "" {
		if t, err := time.Parse(time.RFC3339, e.CreateTime); err == nil {
			m.CreateTime = t
		} else {
			m.CreateTime = time.Now()
		}
	} else {
		m.CreateTime = time.Now()
	}
	m.UpdatedAt = time.Now()

	return m, nil
}

// volumeModelToEntity 将 model.Volume 转换为 entity.EBSVolume
// 此函数为未来 VolumeService 集成 repository 时使用
func volumeModelToEntity(m *model.Volume) (*entity.EBSVolume, error) {
	e := &entity.EBSVolume{
		VolumeID:         m.ID, // 字段名不同，手动映射
		SizeGB:           m.SizeGB,
		SnapshotID:       m.SnapshotID,
		AvailabilityZone: m.AvailabilityZone,
		State:            m.State,
		VolumeType:       m.VolumeType,
		Iops:             m.Iops,
		Encrypted:        m.Encrypted,
		KmsKeyID:         m.KmsKeyID,
	}

	// 处理时间字段
	e.CreateTime = m.CreateTime.Format(time.RFC3339)

	return e, nil
}

// snapshotEntityToModel 将 entity.EBSSnapshot 转换为 model.Snapshot
// 此函数为未来 SnapshotService 集成 repository 时使用
func snapshotEntityToModel(e *entity.EBSSnapshot) (*model.Snapshot, error) {
	m := &model.Snapshot{
		ID:           e.SnapshotID, // 字段名不同，手动映射
		VolumeID:     e.VolumeID,
		State:        e.State,
		Progress:     e.Progress,
		OwnerID:      e.OwnerID,
		Description:  e.Description,
		Encrypted:    e.Encrypted,
		VolumeSizeGB: e.VolumeSizeGB,
	}

	// 处理时间字段
	if e.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			m.StartTime = t
		} else {
			m.StartTime = time.Now()
		}
	} else {
		m.StartTime = time.Now()
	}
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	return m, nil
}

// snapshotModelToEntity 将 model.Snapshot 转换为 entity.EBSSnapshot
// 此函数为未来 SnapshotService 集成 repository 时使用
func snapshotModelToEntity(m *model.Snapshot) (*entity.EBSSnapshot, error) {
	e := &entity.EBSSnapshot{
		SnapshotID:   m.ID, // 字段名不同，手动映射
		VolumeID:     m.VolumeID,
		State:        m.State,
		Progress:     m.Progress,
		OwnerID:      m.OwnerID,
		Description:  m.Description,
		Encrypted:    m.Encrypted,
		VolumeSizeGB: m.VolumeSizeGB,
	}

	// 处理时间字段
	e.StartTime = m.StartTime.Format(time.RFC3339)

	return e, nil
}
