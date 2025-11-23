package metadata

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/rs/zerolog/log"
)

// ==================== Image 元数据管理 ====================

// SaveImage 保存镜像元数据
func (s *LibvirtMetadataStore) SaveImage(ctx context.Context, image *entity.Image) error {
	log.Debug().
		Str("image_id", image.ID).
		Str("name", image.Name).
		Str("state", image.State).
		Msg("Saving image metadata")

	// 1. 构建镜像元数据
	meta := ImageMetadata{
		Version:       "1.0",
		SchemaVersion: "1.0",
		ResourceType:  "image",
		ID:            image.ID,
		Name:          image.Name,
		Description:   image.Description,
		Pool:          image.Pool,
		Path:          image.Path,
		Format:        image.Format,
		State:         image.State,
		CreatedAt:     image.CreatedAt,
		Tags:          make(map[string]string),
	}

	// 2. 获取镜像文件路径
	imagePath := s.getImagePathByID(image.ID)
	if imagePath == "" {
		// 如果镜像文件不存在,使用默认路径或传入的路径
		if image.Path != "" {
			imagePath = image.Path
		} else {
			imagePath = filepath.Join(s.config.BasePath, "images", image.ID+".qcow2")
		}
	}

	// 3. 保存边车元数据文件
	metaPath := getSidecarPath(imagePath)
	if err := saveJSONFile(metaPath, meta); err != nil {
		return fmt.Errorf("save image metadata: %w", err)
	}

	// 4. 更新内存索引
	s.updateImageIndex(image, imagePath)

	log.Info().
		Str("image_id", image.ID).
		Str("meta_path", metaPath).
		Msg("Image metadata saved successfully")

	return nil
}

// GetImage 获取单个镜像
func (s *LibvirtMetadataStore) GetImage(ctx context.Context, imageID string) (*entity.Image, error) {
	log.Debug().Str("image_id", imageID).Msg("Getting image")

	// 1. 从索引获取镜像路径
	imagePath := s.getImagePathByID(imageID)
	if imagePath == "" {
		return nil, fmt.Errorf("image not found: %s", imageID)
	}

	// 2. 读取元数据文件
	metaPath := getSidecarPath(imagePath)
	var meta ImageMetadata
	if err := loadJSONFile(metaPath, &meta); err != nil {
		return nil, fmt.Errorf("load image metadata: %w", err)
	}

	// 3. 获取镜像大小
	sizeGB := uint64(0)
	if info, err := getQCOW2Info(imagePath); err == nil {
		sizeGB = info.VirtualSize / (1024 * 1024 * 1024)
	}

	// 4. 构建镜像对象
	image := &entity.Image{
		ID:          meta.ID,
		Name:        meta.Name,
		Description: meta.Description,
		Pool:        meta.Pool,
		Path:        meta.Path,
		SizeGB:      sizeGB,
		Format:      meta.Format,
		State:       meta.State,
		CreatedAt:   meta.CreatedAt,
	}

	return image, nil
}

// ListImages 列出所有镜像
func (s *LibvirtMetadataStore) ListImages(ctx context.Context) ([]*entity.Image, error) {
	log.Debug().Msg("Listing all images")

	s.index.RLock()
	imageIDs := make([]string, 0, len(s.index.Images))
	for id := range s.index.Images {
		imageIDs = append(imageIDs, id)
	}
	s.index.RUnlock()

	images := make([]*entity.Image, 0, len(imageIDs))
	for _, id := range imageIDs {
		image, err := s.GetImage(ctx, id)
		if err != nil {
			log.Warn().Str("image_id", id).Err(err).Msg("Failed to get image")
			continue
		}
		images = append(images, image)
	}

	log.Debug().Int("count", len(images)).Msg("Listed images")
	return images, nil
}

// DescribeImages 查询镜像(支持过滤)
func (s *LibvirtMetadataStore) DescribeImages(ctx context.Context, req *entity.DescribeImagesRequest) ([]*entity.Image, error) {
	log.Debug().
		Strs("image_ids", req.ImageIDs).
		Interface("filters", req.Filters).
		Msg("Describing images")

	// 如果指定了镜像 ID,直接查询
	if len(req.ImageIDs) > 0 {
		return s.getImagesByIDs(ctx, req.ImageIDs)
	}

	// 否则,使用索引进行过滤查询
	candidateIDs := s.filterImagesByIndex(req)

	// 获取镜像详情
	images := make([]*entity.Image, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		image, err := s.GetImage(ctx, id)
		if err != nil {
			log.Warn().Str("image_id", id).Err(err).Msg("Failed to get image")
			continue
		}

		// 应用额外的过滤条件
		if s.matchesImageFilters(image, req.Filters) {
			images = append(images, image)
		}
	}

	log.Debug().Int("count", len(images)).Msg("Described images")
	return images, nil
}

// DeleteImage 删除镜像元数据
func (s *LibvirtMetadataStore) DeleteImage(ctx context.Context, imageID string) error {
	log.Debug().Str("image_id", imageID).Msg("Deleting image metadata")

	// 1. 获取镜像路径
	imagePath := s.getImagePathByID(imageID)
	if imagePath == "" {
		// 镜像不存在,认为已删除
		return nil
	}

	// 2. 删除边车元数据文件
	metaPath := getSidecarPath(imagePath)
	if err := deleteJSONFile(metaPath); err != nil {
		log.Warn().Err(err).Msg("Failed to delete image metadata file")
	}

	// 3. 从索引中删除
	s.removeImageFromIndex(imageID)

	log.Info().Str("image_id", imageID).Msg("Image metadata deleted")
	return nil
}

// ==================== 辅助函数 ====================

// getImagesByIDs 根据镜像 ID 列表获取镜像
func (s *LibvirtMetadataStore) getImagesByIDs(ctx context.Context, imageIDs []string) ([]*entity.Image, error) {
	images := make([]*entity.Image, 0, len(imageIDs))

	for _, id := range imageIDs {
		image, err := s.GetImage(ctx, id)
		if err != nil {
			log.Warn().Str("image_id", id).Err(err).Msg("Failed to get image")
			continue
		}
		images = append(images, image)
	}

	return images, nil
}

// filterImagesByIndex 使用内存索引过滤镜像
func (s *LibvirtMetadataStore) filterImagesByIndex(req *entity.DescribeImagesRequest) []string {
	s.index.RLock()
	defer s.index.RUnlock()

	// 如果没有过滤条件,返回所有镜像 ID
	if len(req.Filters) == 0 {
		ids := make([]string, 0, len(s.index.Images))
		for id := range s.index.Images {
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
		case "state":
			// 状态过滤
			for _, value := range filter.Values {
				for id, idx := range s.index.Images {
					if idx.State == value {
						matchedIDs = append(matchedIDs, id)
					}
				}
			}

		case "name":
			// 名称过滤
			for _, value := range filter.Values {
				for id, idx := range s.index.Images {
					if idx.Name == value {
						matchedIDs = append(matchedIDs, id)
					}
				}
			}

		case "tag":
			for _, tagPair := range filter.Values {
				if ids, exists := s.index.ImagesByTag[tagPair]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		default:
			continue
		}

		// 合并结果(交集)
		if firstFilter {
			for _, id := range matchedIDs {
				candidateSet[id] = true
			}
			firstFilter = false
		} else {
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

// matchesImageFilters 检查镜像是否匹配过滤条件
func (s *LibvirtMetadataStore) matchesImageFilters(image *entity.Image, filters []entity.Filter) bool {
	for _, filter := range filters {
		switch filter.Name {
		case "image-id":
			matched := false
			for _, value := range filter.Values {
				if image.ID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "format":
			matched := false
			for _, value := range filter.Values {
				if image.Format == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	return true
}

// updateImageIndex 更新镜像索引
func (s *LibvirtMetadataStore) updateImageIndex(image *entity.Image, imagePath string) {
	s.index.Lock()
	defer s.index.Unlock()

	idx := &ImageIndex{
		ID:     image.ID,
		Name:   image.Name,
		Path:   imagePath,
		State:  image.State,
		SizeGB: image.SizeGB,
		
	}

	s.index.Images[image.ID] = idx
}

// removeImageFromIndex 从索引中删除镜像
func (s *LibvirtMetadataStore) removeImageFromIndex(imageID string) {
	s.index.Lock()
	defer s.index.Unlock()

	delete(s.index.Images, imageID)
}
