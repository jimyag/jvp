package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// ==================== 边车文件工具函数 ====================

// loadJSONFile 从 JSON 文件加载数据
func loadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

// saveJSONFile 原子性地保存 JSON 文件
// 使用 temp file + rename 模式保证原子性
func saveJSONFile(path string, v interface{}) error {
	// 1. 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 2. 序列化为 JSON
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	// 3. 写入临时文件
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// 4. 原子性重命名
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // 清理临时文件
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// deleteJSONFile 删除 JSON 文件
func deleteJSONFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getSidecarPath 获取资源的边车文件路径
// 例如: /var/lib/jvp/volumes/vol-123.qcow2 -> /var/lib/jvp/volumes/vol-123.qcow2.jvp.json
func getSidecarPath(resourcePath string) string {
	return resourcePath + ".jvp.json"
}

// getSnapshotIndexPath 获取卷的快照索引文件路径
func getSnapshotIndexPath(basePath, volumeID string) string {
	return filepath.Join(basePath, "volumes", ".snapshots", volumeID+".json")
}

// getKeyPairMetadataPath 获取密钥对元数据文件路径
func getKeyPairMetadataPath(basePath, keyPairID string) string {
	return filepath.Join(basePath, "keypairs", keyPairID+".json")
}

// getKeyPairPublicKeyPath 获取密钥对公钥文件路径
func getKeyPairPublicKeyPath(basePath, keyPairID string) string {
	return filepath.Join(basePath, "keypairs", keyPairID+".pub")
}

// listJSONFiles 列出目录下所有 JSON 文件
//lint:ignore U1000 // 保留供将来使用
func listJSONFiles(dir, pattern string) ([]string, error) {
	fullPattern := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}
	return matches, nil
}

// backupFile 备份文件(用于数据修复)
//
//lint:ignore U1000 // 保留供将来使用
func backupFile(path string) error {
	backupPath := path + ".backup"

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	log.Debug().
		Str("original", path).
		Str("backup", backupPath).
		Msg("File backed up")

	return nil
}

// restoreFromBackup 从备份恢复文件
func restoreFromBackup(path string) error {
	backupPath := path + ".backup"

	if !fileExists(backupPath) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	log.Info().
		Str("path", path).
		Str("backup", backupPath).
		Msg("File restored from backup")

	return nil
}

// validateJSONFile 验证 JSON 文件是否有效
func validateJSONFile(path string, v interface{}) error {
	if !fileExists(path) {
		return fmt.Errorf("file not found: %s", path)
	}

	if err := loadJSONFile(path, v); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return nil
}
