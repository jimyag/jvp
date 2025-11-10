package idgen

import (
	"fmt"
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

// Generator 递增 ID 生成器
// 使用 Sonyflake 算法生成全局唯一且递增的 ID
type Generator struct {
	sf *sonyflake.Sonyflake
}

var (
	defaultGenerator     *Generator
	defaultGeneratorOnce sync.Once
)

// initDefaultGenerator 初始化默认生成器
func initDefaultGenerator() {
	defaultGenerator = New()
}

// DefaultGenerator 返回默认的 ID 生成器
func DefaultGenerator() *Generator {
	defaultGeneratorOnce.Do(initDefaultGenerator)
	return defaultGenerator
}

// New 创建新的 ID 生成器
func New() *Generator {
	// 使用默认设置创建 Sonyflake
	// 如果需要自定义机器 ID，可以通过 Settings 配置
	sf := sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // 起始时间
	})
	if sf == nil {
		// 如果创建失败，使用当前时间作为起始时间
		sf = sonyflake.NewSonyflake(sonyflake.Settings{
			StartTime: time.Now(),
		})
	}

	return &Generator{
		sf: sf,
	}
}

// generateIDWithPrefix 生成带前缀的 ID
func (g *Generator) generateIDWithPrefix(prefix, errorMsg string) (string, error) {
	id, err := g.sf.NextID()
	if err != nil {
		return "", fmt.Errorf("%s: %w", errorMsg, err)
	}
	return fmt.Sprintf("%s-%d", prefix, id), nil
}

// GenerateImageID 生成镜像 ID（格式：ami-{递增 ID}）
func (g *Generator) GenerateImageID() (string, error) {
	return g.generateIDWithPrefix("ami", "generate image ID")
}

// GenerateVolumeID 生成 Volume ID（格式：vol-{递增 ID}）
func (g *Generator) GenerateVolumeID() (string, error) {
	return g.generateIDWithPrefix("vol", "generate volume ID")
}

// GenerateInstanceID 生成 Instance ID（格式：i-{递增 ID}）
func (g *Generator) GenerateInstanceID() (string, error) {
	return g.generateIDWithPrefix("i", "generate instance ID")
}

// GenerateSnapshotID 生成 Snapshot ID（格式：snap-{递增 ID}）
func (g *Generator) GenerateSnapshotID() (string, error) {
	return g.generateIDWithPrefix("snap", "generate snapshot ID")
}

// GenerateID 生成通用递增 ID
func (g *Generator) GenerateID() (uint64, error) {
	return g.sf.NextID()
}

// 包级别的便捷函数，使用默认生成器

// GenerateImageID 使用默认生成器生成镜像 ID
func GenerateImageID() (string, error) {
	return DefaultGenerator().GenerateImageID()
}

// GenerateVolumeID 使用默认生成器生成 Volume ID
func GenerateVolumeID() (string, error) {
	return DefaultGenerator().GenerateVolumeID()
}

// GenerateInstanceID 使用默认生成器生成 Instance ID
func GenerateInstanceID() (string, error) {
	return DefaultGenerator().GenerateInstanceID()
}

// GenerateSnapshotID 使用默认生成器生成 Snapshot ID
func GenerateSnapshotID() (string, error) {
	return DefaultGenerator().GenerateSnapshotID()
}

// GenerateID 使用默认生成器生成通用递增 ID
func GenerateID() (uint64, error) {
	return DefaultGenerator().GenerateID()
}
