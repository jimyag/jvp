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
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/pkg/apierror"
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
	imageRepo      repository.ImageRepository

	// 镜像存储配置
	imagesPoolName string
	imagesPoolPath string
}

// NewImageService 创建新的 Image Service
func NewImageService(
	storageService *StorageService,
	libvirtClient libvirt.LibvirtClient,
	repo *repository.Repository,
) (*ImageService, error) {
	return &ImageService{
		storageService: storageService,
		libvirtClient:  libvirtClient,
		qemuImgClient:  qemuimg.New(""),
		idGen:          idgen.New(),
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // 下载镜像可能需要较长时间
		},
		imageRepo:      repository.NewImageRepository(repo.DB()),
		imagesPoolName: "images",
		imagesPoolPath: "/var/lib/jvp/images/images",
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

	// 保存到数据库
	imageModel, err := imageEntityToModel(image)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert image to model", err)
	}
	if err := s.imageRepo.Create(ctx, imageModel); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save image to database", err)
	}
	logger.Info().Str("image_id", imageID).Msg("Image saved to database")

	return image, nil
}

// GetImage 获取镜像信息
func (s *ImageService) GetImage(ctx context.Context, imageID string) (*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("image_id", imageID).Msg("Getting image")

	// 优先从数据库查询
	imageModel, err := s.imageRepo.GetByID(ctx, imageID)
	if err == nil {
		return imageModelToEntity(imageModel)
	}

	// 如果数据库中没有，检查是否是默认镜像（兼容旧数据）
	if defaultImg := s.GetDefaultImageByName(imageID); defaultImg != nil {
		imagePath := filepath.Join(s.imagesPoolPath, defaultImg.Filename)
		fileInfo, err := os.Stat(imagePath)
		if err != nil {
			// 如果镜像不存在，尝试下载
			logger.Info().
				Str("image_id", imageID).
				Str("url", defaultImg.URL).
				Msg("Default image not found, downloading...")

			_, err := s.DownloadImage(ctx, defaultImg.URL, defaultImg.Filename)
			if err != nil {
				return nil, fmt.Errorf("download default image: %w", err)
			}

			// 重新获取文件信息
			fileInfo, err = os.Stat(imagePath)
			if err != nil {
				return nil, fmt.Errorf("get image file info after download: %w", err)
			}
		}

		sizeGB := uint64(fileInfo.Size() / (1024 * 1024 * 1024))
		if sizeGB == 0 {
			sizeGB = 1
		}

		return &entity.Image{
			ID:          imageID,
			Name:        defaultImg.DisplayName,
			Description: fmt.Sprintf("Default Ubuntu image: %s", defaultImg.DisplayName),
			Pool:        s.imagesPoolName,
			Path:        imagePath,
			SizeGB:      sizeGB,
			Format:      "qcow2",
			State:       "available",
			CreatedAt:   fileInfo.ModTime().Format(time.RFC3339),
		}, nil
	}

	// 根据 imageID 查找镜像文件
	// 镜像文件命名：ami-{uuid}.qcow2
	imagePath := filepath.Join(s.imagesPoolPath, imageID+".qcow2")

	if _, err := os.Stat(imagePath); err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	// 获取镜像信息
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	sizeGB := uint64(fileInfo.Size() / (1024 * 1024 * 1024))
	if sizeGB == 0 {
		sizeGB = 1
	}

	return &entity.Image{
		ID:        imageID,
		Pool:      s.imagesPoolName,
		Path:      imagePath,
		SizeGB:    sizeGB,
		Format:    "qcow2",
		State:     "available",
		CreatedAt: fileInfo.ModTime().Format(time.RFC3339),
	}, nil
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
	logger.Info().Msg("Listing images")

	// 从数据库查询
	filters := make(map[string]interface{})
	imageModels, err := s.imageRepo.List(ctx, filters)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list images from database")
		return nil, fmt.Errorf("list images from database: %w", err)
	}

	logger.Info().
		Int("totalInDB", len(imageModels)).
		Msg("Retrieved images from database")

	images := make([]*entity.Image, 0, len(imageModels))
	for _, imageModel := range imageModels {
		image, err := imageModelToEntity(imageModel)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("image_id", imageModel.ID).
				Msg("Failed to convert image model to entity, skipping")
			continue
		}
		images = append(images, image)
	}

	logger.Info().
		Int("totalInDB", len(imageModels)).
		Int("converted", len(images)).
		Msg("Converted images from database")

	// 同时添加默认镜像（如果已下载但未注册到数据库）
	// 确保目录存在
	if err := os.MkdirAll(s.imagesPoolPath, 0o755); err != nil {
		return nil, fmt.Errorf("create images directory: %w", err)
	}

	// 首先添加默认镜像（如果已下载但未在数据库中）
	for _, defaultImg := range DefaultImages {
		// 检查是否已在数据库中
		exists := false
		for _, img := range images {
			if img.ID == defaultImg.Name {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		savePath := filepath.Join(s.imagesPoolPath, defaultImg.Filename)
		if fileInfo, err := os.Stat(savePath); err == nil {
			sizeGB := uint64(fileInfo.Size() / (1024 * 1024 * 1024))
			if sizeGB == 0 {
				sizeGB = 1
			}

			// 使用镜像名称作为 ID（如：ubuntu-jammy）
			images = append(images, &entity.Image{
				ID:          defaultImg.Name,
				Name:        defaultImg.DisplayName,
				Description: fmt.Sprintf("Default Ubuntu image: %s", defaultImg.DisplayName),
				Pool:        s.imagesPoolName,
				Path:        savePath,
				SizeGB:      sizeGB,
				Format:      "qcow2",
				State:       "available",
				CreatedAt:   fileInfo.ModTime().Format(time.RFC3339),
			})
		}
	}

	return images, nil
}

// DeleteImage 删除镜像
func (s *ImageService) DeleteImage(ctx context.Context, imageID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("image_id", imageID).Msg("Deleting image")

	// 从数据库查询镜像信息
	imageModel, err := s.imageRepo.GetByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("get image from database: %w", err)
	}

	// 删除文件
	if err := os.Remove(imageModel.Path); err != nil {
		logger.Warn().Err(err).Str("path", imageModel.Path).Msg("Failed to delete image file")
		// 继续执行，即使文件删除失败也继续删除数据库记录
	}

	// 从数据库删除（软删除）
	if err := s.imageRepo.Delete(ctx, imageID); err != nil {
		return fmt.Errorf("delete image from database: %w", err)
	}

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
