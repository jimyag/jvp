// Package repository 提供数据持久化层实现
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，不需要 CGO
)

// Repository 数据库仓库
type Repository struct {
	db *gorm.DB
}

// New 创建新的 Repository 实例
func New(dbPath string) (*Repository, error) {
	// 确保数据库目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// 连接数据库（使用纯 Go SQLite 驱动，不需要 CGO）
	// 直接使用 database/sql + modernc.org/sqlite 创建连接，然后传递给 GORM
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 使用 GORM 的 Dialector 包装已创建的 sql.DB 连接
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
		Conn:       sqlDB,
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 生产环境可以设置为 Silent
	})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("open gorm database: %w", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.Image{},
		&model.Volume{},
		&model.Snapshot{},
		&model.VolumeAttachment{},
		&model.Tag{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// 创建索引（GORM 的 AutoMigrate 可能不会创建所有索引，手动确保）
	if err := createIndexes(db); err != nil {
		return nil, fmt.Errorf("create indexes: %w", err)
	}

	return &Repository{db: db}, nil
}

// DB 返回 GORM 数据库实例（用于 Repository 实现）
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// WithContext 返回带上下文的数据库实例
func (r *Repository) WithContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// Close 关闭数据库连接
func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// createIndexes 创建额外的索引和唯一约束
func createIndexes(db *gorm.DB) error {
	// volume_attachments 表的唯一约束（一个卷只能附加到一个实例）
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_volume_instance_unique 
		ON volume_attachments(volume_id, instance_id)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("create unique index on volume_attachments: %w", err)
	}

	// tags 表的唯一约束（同一资源的同一 key 只能有一个值）
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_unique 
		ON tags(resource_type, resource_id, tag_key)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("create unique index on tags: %w", err)
	}

	return nil
}
