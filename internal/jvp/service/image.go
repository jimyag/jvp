// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service 和 Image Service，用于管理存储资源和镜像模板
package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// DefaultImage 默认镜像配置
type DefaultImage struct {
	Name        string // 镜像名称（如：ubuntu-jammy）
	DisplayName string // 显示名称（如：Ubuntu 22.04 LTS (Jammy Jellyfish)）
	URL         string // 下载 URL
	Filename    string // 保存的文件名（如：ubuntu-jammy-server-cloudimg-amd64.img）
}

// DefaultImages 默认镜像列表
var DefaultImages = []DefaultImage{
	{
		Name:        "ubuntu-jammy",
		DisplayName: "Ubuntu 22.04 LTS (Jammy Jellyfish)",
		URL:         "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		Filename:    "ubuntu-jammy-server-cloudimg-amd64.img",
	},
	{
		Name:        "ubuntu-focal",
		DisplayName: "Ubuntu 20.04 LTS (Focal Fossa)",
		URL:         "https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-amd64.img",
		Filename:    "ubuntu-focal-server-cloudimg-amd64.img",
	},
	{
		Name:        "ubuntu-noble",
		DisplayName: "Ubuntu 24.04 LTS (Noble Numbat)",
		URL:         "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		Filename:    "ubuntu-noble-server-cloudimg-amd64.img",
	},
}

// ImageService 镜像服务，管理镜像模板
type ImageService struct {
	storageService *StorageService
	libvirtClient  libvirt.LibvirtClient
	qemuImgClient  qemuimg.QemuImgClient
	idGen          *idgen.Generator
	httpClient     *http.Client

	// 镜像存储配置
	imagesPoolName string
	imagesPoolPath string
}

// NewImageService 创建新的 Image Service
func NewImageService(
	storageService *StorageService,
	libvirtClient libvirt.LibvirtClient,
) (*ImageService, error) {
	return &ImageService{
		storageService: storageService,
		libvirtClient:  libvirtClient,
		qemuImgClient:  qemuimg.New(""),
		idGen:          idgen.New(),
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // 下载镜像可能需要较长时间
		},
		imagesPoolName: storageService.imagesPoolName,
		imagesPoolPath: storageService.imagesPoolPath,
	}, nil
}

// RegisterImage 注册镜像（镜像文件必须已存在于 images pool 中）
func (s *ImageService) RegisterImage(ctx context.Context, req *entity.RegisterImageRequest) (*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("name", req.Name).
		Str("path", req.Path).
		Msg("Registering image")

	// 验证镜像文件是否存在
	if _, err := os.Stat(req.Path); err != nil {
		return nil, fmt.Errorf("image file not found: %w", err)
	}

	// 从文件大小获取镜像大小
	fileInfo, err := os.Stat(req.Path)
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	sizeGB := uint64(fileInfo.Size() / (1024 * 1024 * 1024))
	if sizeGB == 0 {
		sizeGB = 2 // 至少 2GB
	}

	// 生成镜像 ID
	imageID, err := s.idGen.GenerateImageID()
	if err != nil {
		return nil, fmt.Errorf("generate image ID: %w", err)
	}

	image := &entity.Image{
		ID:          imageID,
		Name:        req.Name,
		Description: req.Description,
		Pool:        s.imagesPoolName,
		Path:        req.Path,
		SizeGB:      sizeGB,
		Format:      "qcow2", // 默认格式
		State:       "available",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	logger.Info().
		Str("image_id", imageID).
		Str("name", req.Name).
		Msg("Image registered successfully")

	// 镜像文件已存储在 images pool 中，不需要额外保存元数据

	return image, nil
}

// GetImage 获取镜像信息
func (s *ImageService) GetImage(ctx context.Context, imageID string) (*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("image_id", imageID).Msg("Getting image")

	// 从 images storage pool 查询所有镜像
	images, err := s.ListImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	// 查找匹配的镜像
	for _, img := range images {
		if img != nil && img.ID == imageID {
			return img, nil
		}
	}

	return nil, fmt.Errorf("image not found: %s", imageID)
}

// DescribeImages 描述镜像（支持过滤和分页）
func (s *ImageService) DescribeImages(ctx context.Context, req *entity.DescribeImagesRequest) ([]entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Describing images")

	var images []entity.Image

	if len(req.ImageIDs) > 0 {
		// 查询指定的镜像
		for _, imageID := range req.ImageIDs {
			image, err := s.GetImage(ctx, imageID)
			if err != nil {
				// 如果镜像不存在，跳过
				logger.Warn().
					Str("imageID", imageID).
					Err(err).
					Msg("Image not found, skipping")
				continue
			}
			images = append(images, *image)
		}

		logger.Info().
			Int("requested", len(req.ImageIDs)).
			Int("found", len(images)).
			Msg("Describe images by IDs completed")
	} else {
		// 列出所有镜像
		imageList, err := s.ListImages(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to list images")
			return nil, fmt.Errorf("list images: %w", err)
		}

		for _, img := range imageList {
			images = append(images, *img)
		}

		logger.Info().
			Int("total", len(images)).
			Msg("Describe all images completed")
	}

	// TODO: 应用过滤器和分页

	return images, nil
}

// ListImages 列出所有镜像
func (s *ImageService) ListImages(ctx context.Context) ([]*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Listing images from storage pool")

	// 确保 images pool 存在
	if err := s.storageService.EnsurePool(ctx, s.imagesPoolName); err != nil {
		return nil, fmt.Errorf("ensure images pool: %w", err)
	}

	// 从 images storage pool 查询所有卷
	volumes, err := s.storageService.ListVolumes(ctx, s.imagesPoolName)
	if err != nil {
		return nil, fmt.Errorf("list volumes in images pool: %w", err)
	}

	var images []*entity.Image
	for _, vol := range volumes {
		// 使用卷名作为镜像 ID（去掉 .qcow2 扩展名）
		imageID := vol.Name
		if len(imageID) > 6 && imageID[len(imageID)-6:] == ".qcow2" {
			imageID = imageID[:len(imageID)-6]
		}

		// 检查是否是默认镜像
		displayName := imageID
		description := ""
		if defaultImg := s.GetDefaultImageByName(imageID); defaultImg != nil {
			displayName = defaultImg.DisplayName
			description = fmt.Sprintf("Default Ubuntu image: %s", defaultImg.DisplayName)
		}

		image := &entity.Image{
			ID:          imageID,
			Name:        displayName,
			Description: description,
			Pool:        s.imagesPoolName,
			Path:        vol.Path,
			SizeGB:      vol.CapacityB / (1024 * 1024 * 1024),
			Format:      "qcow2",
			State:       "available",
			CreatedAt:   time.Now().Format(time.RFC3339),
		}
		images = append(images, image)
	}

	logger.Info().
		Int("total", len(images)).
		Msg("Retrieved images from storage pool")

	return images, nil
}

// DeleteImage 删除镜像
func (s *ImageService) DeleteImage(ctx context.Context, imageID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("image_id", imageID).Msg("Deleting image")

	// 从 images pool 获取镜像信息
	image, err := s.GetImage(ctx, imageID)
	if err != nil {
		return fmt.Errorf("get image: %w", err)
	}

	// 删除文件
	if err := os.Remove(image.Path); err != nil {
		logger.Warn().Err(err).Str("path", image.Path).Msg("Failed to delete image file")
		return fmt.Errorf("delete image file: %w", err)
	}

	// 镜像文件已删除，libvirt storage pool 会自动更新

	logger.Info().Str("image_id", imageID).Msg("Image deleted successfully")
	return nil
}

// CreateImageFromInstance 从 Instance 创建镜像
func (s *ImageService) CreateImageFromInstance(
	ctx context.Context,
	req *entity.CreateImageFromInstanceRequest,
) (*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instance_id", req.InstanceID).
		Str("image_name", req.ImageName).
		Msg("Creating image from instance")

	// 1. 获取 Instance 的磁盘路径
	// 通过 libvirt 获取 domain 的磁盘信息
	disks, err := s.libvirtClient.GetDomainDisks(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get instance disks: %w", err)
	}

	if len(disks) == 0 || disks[0].Source.File == "" {
		return nil, fmt.Errorf("instance %s has no disk", req.InstanceID)
	}

	sourceDiskPath := disks[0].Source.File

	// 2. 生成镜像 ID
	imageID, err := s.idGen.GenerateImageID()
	if err != nil {
		return nil, fmt.Errorf("generate image ID: %w", err)
	}

	// 3. 确定镜像文件路径（保存到 images pool）
	imageFilename := imageID + ".qcow2"
	imagePath := filepath.Join(s.imagesPoolPath, imageFilename)

	// 确保 images pool 存在
	if err := s.storageService.EnsurePool(ctx, s.imagesPoolName); err != nil {
		return nil, fmt.Errorf("ensure images pool: %w", err)
	}

	// 4. 使用 qemu-img convert 创建镜像（从实例磁盘转换为镜像）
	logger.Info().
		Str("source", sourceDiskPath).
		Str("target", imagePath).
		Msg("Converting disk to image")

	err = s.qemuImgClient.Convert(ctx, "qcow2", "qcow2", sourceDiskPath, imagePath)
	if err != nil {
		return nil, fmt.Errorf("convert disk to image: %w", err)
	}

	// 5. 获取镜像大小
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		// 清理已创建的镜像文件
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("get image file info: %w", err)
	}

	sizeGB := uint64(fileInfo.Size() / (1024 * 1024 * 1024))
	if sizeGB == 0 {
		sizeGB = 1
	}

	// 6. 修复文件权限（确保 libvirt-qemu 可以访问）
	// 获取文件信息以确定是否需要修复权限
	_ = fileInfo

	// 7. 注册镜像
	registerReq := &entity.RegisterImageRequest{
		Name:        req.ImageName,
		Description: req.Description,
		Path:        imagePath,
		Pool:        s.imagesPoolName,
	}

	image, err := s.RegisterImage(ctx, registerReq)
	if err != nil {
		// 清理已创建的镜像文件
		_ = os.Remove(imagePath)
		return nil, fmt.Errorf("register image: %w", err)
	}

	// 使用生成的 imageID
	image.ID = imageID

	logger.Info().
		Str("image_id", imageID).
		Str("instance_id", req.InstanceID).
		Msg("Image created from instance successfully")

	return image, nil
}

// DownloadImage 下载镜像文件
func (s *ImageService) DownloadImage(ctx context.Context, url, filename string) (string, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("url", url).
		Str("filename", filename).
		Msg("Downloading image")

	// 确保目录存在
	if err := os.MkdirAll(s.imagesPoolPath, 0o755); err != nil {
		return "", fmt.Errorf("create images directory: %w", err)
	}

	// 构建保存路径
	savePath := filepath.Join(s.imagesPoolPath, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(savePath); err == nil {
		logger.Info().
			Str("path", savePath).
			Msg("Image file already exists, skipping download")
		return savePath, nil
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// 执行下载
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 创建临时文件
	tmpPath := savePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer out.Close()

	// 复制数据并显示进度
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	// 重命名为最终文件
	if err := os.Rename(tmpPath, savePath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("rename file: %w", err)
	}

	logger.Info().
		Str("path", savePath).
		Int64("size_bytes", written).
		Msg("Image downloaded successfully")

	return savePath, nil
}

// EnsureDefaultImages 确保默认镜像存在（如果不存在则下载）
func (s *ImageService) EnsureDefaultImages(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Int("total", len(DefaultImages)).
		Msg("Ensuring default images exist")

	downloadedCount := 0
	existingCount := 0

	for _, img := range DefaultImages {
		// 检查文件是否已存在
		savePath := filepath.Join(s.imagesPoolPath, img.Filename)
		if _, err := os.Stat(savePath); err == nil {
			logger.Info().
				Str("name", img.Name).
				Str("display_name", img.DisplayName).
				Str("path", savePath).
				Msg("Default image already exists")
			existingCount++
			continue
		}

		// 下载镜像
		logger.Info().
			Str("name", img.Name).
			Str("display_name", img.DisplayName).
			Str("url", img.URL).
			Msg("Downloading default image")

		_, err := s.DownloadImage(ctx, img.URL, img.Filename)
		if err != nil {
			logger.Error().
				Err(err).
				Str("name", img.Name).
				Msg("Failed to download default image")
			return fmt.Errorf("download %s: %w", img.Name, err)
		}

		logger.Info().
			Str("name", img.Name).
			Str("display_name", img.DisplayName).
			Str("path", savePath).
			Msg("Default image downloaded successfully")
		downloadedCount++
	}

	logger.Info().
		Int("total", len(DefaultImages)).
		Int("downloaded", downloadedCount).
		Int("existing", existingCount).
		Msg("All default images are ready")

	return nil
}

// GetDefaultImageByName 根据名称获取默认镜像信息
func (s *ImageService) GetDefaultImageByName(name string) *DefaultImage {
	for _, img := range DefaultImages {
		if img.Name == name {
			return &img
		}
	}
	return nil
}

// ListDefaultImages 列出所有默认镜像信息
func (s *ImageService) ListDefaultImages() []DefaultImage {
	return DefaultImages
}
