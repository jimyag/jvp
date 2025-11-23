package metadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// FileLock 文件锁结构
type FileLock struct {
	path    string
	file    *os.File
	timeout time.Duration
}

// NewFileLock 创建文件锁
func NewFileLock(lockDir, resourceID string, timeout time.Duration) *FileLock {
	lockPath := filepath.Join(lockDir, resourceID+".lock")
	return &FileLock{
		path:    lockPath,
		timeout: timeout,
	}
}

// Lock 获取锁
func (fl *FileLock) Lock(ctx context.Context) error {
	// 确保锁目录存在
	lockDir := filepath.Dir(fl.path)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}

	// 打开或创建锁文件
	file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	fl.file = file

	// 尝试获取文件锁(带超时)
	deadline := time.Now().Add(fl.timeout)
	for {
		// 尝试获取独占锁
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// 成功获取锁
			log.Debug().Str("lock_path", fl.path).Msg("Lock acquired")
			return nil
		}

		// 检查是否超时
		if time.Now().After(deadline) {
			file.Close()
			fl.file = nil
			return fmt.Errorf("lock timeout after %v", fl.timeout)
		}

		// 检查 context 是否取消
		select {
		case <-ctx.Done():
			file.Close()
			fl.file = nil
			return ctx.Err()
		default:
		}

		// 等待一小段时间后重试
		time.Sleep(100 * time.Millisecond)
	}
}

// Unlock 释放锁
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	// 释放文件锁
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		log.Warn().Err(err).Str("lock_path", fl.path).Msg("Failed to unlock file")
	}

	// 关闭文件
	if err := fl.file.Close(); err != nil {
		log.Warn().Err(err).Str("lock_path", fl.path).Msg("Failed to close lock file")
	}

	fl.file = nil

	log.Debug().Str("lock_path", fl.path).Msg("Lock released")
	return nil
}

// TryLock 尝试获取锁(非阻塞)
func (fl *FileLock) TryLock() error {
	// 确保锁目录存在
	lockDir := filepath.Dir(fl.path)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}

	// 打开或创建锁文件
	file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	// 尝试获取独占锁(非阻塞)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return fmt.Errorf("lock busy: %w", err)
	}

	fl.file = file
	log.Debug().Str("lock_path", fl.path).Msg("Lock acquired (non-blocking)")
	return nil
}

// IsLocked 检查资源是否被锁定
func IsLocked(lockDir, resourceID string) bool {
	lockPath := filepath.Join(lockDir, resourceID+".lock")

	// 检查锁文件是否存在
	if !fileExists(lockPath) {
		return false
	}

	// 尝试获取共享锁(用于检查)
	file, err := os.Open(lockPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 尝试获取共享锁
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err != nil {
		// 无法获取共享锁,说明有独占锁
		return true
	}

	// 释放共享锁
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false
}

// WithLock 使用锁执行函数
func WithLock(ctx context.Context, lockDir, resourceID string, timeout time.Duration, fn func() error) error {
	lock := NewFileLock(lockDir, resourceID, timeout)

	// 获取锁
	if err := lock.Lock(ctx); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer lock.Unlock()

	// 执行函数
	return fn()
}

// WithLockRetry 带重试的锁执行
func WithLockRetry(ctx context.Context, lockDir, resourceID string, timeout time.Duration, maxRetries int, fn func() error) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := WithLock(ctx, lockDir, resourceID, timeout, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是上下文取消,直接返回
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 等待后重试
		if i < maxRetries-1 {
			log.Debug().
				Str("resource_id", resourceID).
				Int("attempt", i+1).
				Err(err).
				Msg("Lock operation failed, retrying")

			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	return fmt.Errorf("lock operation failed after %d retries: %w", maxRetries, lastErr)
}

// LockResource 锁定资源(用于 MetadataStore)
func (s *LibvirtMetadataStore) LockResource(ctx context.Context, resourceType, resourceID string) (*FileLock, error) {
	lockDir := filepath.Join(s.config.BasePath, "locks", resourceType)
	lock := NewFileLock(lockDir, resourceID, s.config.LockTimeout)

	if err := lock.Lock(ctx); err != nil {
		return nil, err
	}

	return lock, nil
}

// WithResourceLock 使用资源锁执行函数
func (s *LibvirtMetadataStore) WithResourceLock(ctx context.Context, resourceType, resourceID string, fn func() error) error {
	lockDir := filepath.Join(s.config.BasePath, "locks", resourceType)
	return WithLock(ctx, lockDir, resourceID, s.config.LockTimeout, fn)
}
