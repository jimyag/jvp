package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

// DownloadTaskStatus 下载任务状态
type DownloadTaskStatus string

const (
	DownloadTaskStatusPending   DownloadTaskStatus = "pending"
	DownloadTaskStatusRunning   DownloadTaskStatus = "running"
	DownloadTaskStatusCompleted DownloadTaskStatus = "completed"
	DownloadTaskStatusFailed    DownloadTaskStatus = "failed"
)

// DownloadTask 下载任务
type DownloadTask struct {
	ID         string             `json:"id"`
	NodeName   string             `json:"node_name"`
	PoolName   string             `json:"pool_name"`
	VolumeName string             `json:"volume_name"`
	URL        string             `json:"url"`
	Status     DownloadTaskStatus `json:"status"`
	Error      string             `json:"error,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// DownloadTaskManager 下载任务管理器
type DownloadTaskManager struct {
	mu            sync.RWMutex
	tasks         map[string]*DownloadTask // key: taskID
	tasksByVolume map[string]string        // key: nodeName:poolName:volumeName -> taskID
}

// NewDownloadTaskManager 创建下载任务管理器
func NewDownloadTaskManager() *DownloadTaskManager {
	return &DownloadTaskManager{
		tasks:         make(map[string]*DownloadTask),
		tasksByVolume: make(map[string]string),
	}
}

// volumeKey 生成 volume 唯一标识
func volumeKey(nodeName, poolName, volumeName string) string {
	return fmt.Sprintf("%s:%s:%s", nodeName, poolName, volumeName)
}

// GetTaskByVolume 根据 volume 信息获取任务
func (m *DownloadTaskManager) GetTaskByVolume(nodeName, poolName, volumeName string) *DownloadTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := volumeKey(nodeName, poolName, volumeName)
	taskID, exists := m.tasksByVolume[key]
	if !exists {
		return nil
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil
	}

	// 返回副本
	taskCopy := *task
	return &taskCopy
}

// GetTask 根据任务 ID 获取任务
func (m *DownloadTaskManager) GetTask(taskID string) *DownloadTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil
	}

	// 返回副本
	taskCopy := *task
	return &taskCopy
}

// CreateTask 创建下载任务
// 如果已有相同 volume 的任务在运行中，返回现有任务
func (m *DownloadTaskManager) CreateTask(taskID, nodeName, poolName, volumeName, url string) (*DownloadTask, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := volumeKey(nodeName, poolName, volumeName)

	// 检查是否已有任务
	if existingTaskID, exists := m.tasksByVolume[key]; exists {
		existingTask := m.tasks[existingTaskID]
		// 如果任务正在运行中，返回现有任务
		if existingTask.Status == DownloadTaskStatusPending || existingTask.Status == DownloadTaskStatusRunning {
			taskCopy := *existingTask
			return &taskCopy, false // false 表示不是新创建的
		}
		// 如果任务已完成或失败，删除旧任务
		delete(m.tasks, existingTaskID)
		delete(m.tasksByVolume, key)
	}

	// 创建新任务
	now := time.Now()
	task := &DownloadTask{
		ID:         taskID,
		NodeName:   nodeName,
		PoolName:   poolName,
		VolumeName: volumeName,
		URL:        url,
		Status:     DownloadTaskStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	m.tasks[taskID] = task
	m.tasksByVolume[key] = taskID

	taskCopy := *task
	return &taskCopy, true // true 表示新创建的
}

// UpdateTaskStatus 更新任务状态
func (m *DownloadTaskManager) UpdateTaskStatus(taskID string, status DownloadTaskStatus, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return
	}

	task.Status = status
	task.Error = errMsg
	task.UpdatedAt = time.Now()
}

// RemoveTask 移除任务
func (m *DownloadTaskManager) RemoveTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return
	}

	key := volumeKey(task.NodeName, task.PoolName, task.VolumeName)
	delete(m.tasksByVolume, key)
	delete(m.tasks, taskID)
}

// CleanupOldTasks 清理已完成的旧任务（超过指定时间）
func (m *DownloadTaskManager) CleanupOldTasks(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for taskID, task := range m.tasks {
		// 只清理已完成或失败的任务
		if task.Status == DownloadTaskStatusCompleted || task.Status == DownloadTaskStatusFailed {
			if now.Sub(task.UpdatedAt) > maxAge {
				key := volumeKey(task.NodeName, task.PoolName, task.VolumeName)
				delete(m.tasksByVolume, key)
				delete(m.tasks, taskID)
			}
		}
	}
}

// ListActiveTasks 列出所有活跃的下载任务（pending 或 running）
func (m *DownloadTaskManager) ListActiveTasks() []*DownloadTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*DownloadTask
	for _, task := range m.tasks {
		if task.Status == DownloadTaskStatusPending || task.Status == DownloadTaskStatusRunning {
			taskCopy := *task
			result = append(result, &taskCopy)
		}
	}
	return result
}

// StartDownload 启动异步下载
// 下载文件到存储池根目录（作为存储卷）
// 模板元数据会在下载完成后单独保存到 _templates_ 目录
func (m *DownloadTaskManager) StartDownload(
	ctx context.Context,
	task *DownloadTask,
	client libvirt.LibvirtClient,
	onComplete func(task *DownloadTask, err error),
) {
	go func() {
		logger := zerolog.Ctx(ctx)

		// 更新状态为运行中
		m.UpdateTaskStatus(task.ID, DownloadTaskStatusRunning, "")

		logger.Info().
			Str("task_id", task.ID).
			Str("url", task.URL).
			Str("volume_name", task.VolumeName).
			Msg("Starting download task")

		// 执行下载到存储池的 _templates_ 目录
		err := downloadToTemplatesDir(client, task.PoolName, task.VolumeName, task.URL)

		if err != nil {
			logger.Error().
				Err(err).
				Str("task_id", task.ID).
				Msg("Download task failed")
			m.UpdateTaskStatus(task.ID, DownloadTaskStatusFailed, err.Error())
		} else {
			logger.Info().
				Str("task_id", task.ID).
				Msg("Download task completed")
			m.UpdateTaskStatus(task.ID, DownloadTaskStatusCompleted, "")
		}

		// 调用完成回调
		if onComplete != nil {
			updatedTask := m.GetTask(task.ID)
			onComplete(updatedTask, err)
		}
	}()
}

// downloadToPool 下载文件到存储池（普通卷，存储池根目录）
// 通过 libvirt 的基础接口实现下载功能
func downloadToPool(client libvirt.LibvirtClient, poolName, volumeName, downloadURL string) error {
	return downloadToDir(client, poolName, "", volumeName, downloadURL)
}

// downloadToTemplatesDir 下载模板文件到存储池的 _templates_ 目录
func downloadToTemplatesDir(client libvirt.LibvirtClient, poolName, fileName, downloadURL string) error {
	return downloadToDir(client, poolName, TemplatesDirName, fileName, downloadURL)
}

// downloadToDir 下载文件到存储池的指定子目录
func downloadToDir(client libvirt.LibvirtClient, poolName, subDir, fileName, downloadURL string) error {
	// 获取存储池信息
	poolInfo, err := client.GetStoragePool(poolName)
	if err != nil {
		return fmt.Errorf("get storage pool: %w", err)
	}

	if poolInfo.Path == "" {
		return fmt.Errorf("storage pool %s has no target path", poolName)
	}

	// 构建目标目录和文件路径
	var targetDir, targetPath string
	if subDir == "" {
		targetDir = poolInfo.Path
		targetPath = poolInfo.Path + "/" + fileName
	} else {
		targetDir = poolInfo.Path + "/" + subDir
		targetPath = targetDir + "/" + fileName
	}

	// 根据是否是远程连接选择下载方式
	if client.IsRemoteConnection() {
		// 远程连接：先创建目录，然后通过 SSH 执行下载命令
		if subDir != "" {
			mkdirCmd := fmt.Sprintf("mkdir -p '%s'", targetDir)
			if err := client.ExecuteRemoteCommand(mkdirCmd); err != nil {
				return fmt.Errorf("create directory via SSH: %w", err)
			}
		}

		downloadCmd := fmt.Sprintf(
			`command -v wget >/dev/null 2>&1 && wget -q -O '%s' '%s' || curl -sSL -o '%s' '%s'`,
			targetPath, downloadURL, targetPath, downloadURL,
		)
		if err := client.ExecuteRemoteCommand(downloadCmd); err != nil {
			return fmt.Errorf("download via SSH: %w", err)
		}
	} else {
		// 本地连接：先创建目录，然后下载
		if subDir != "" {
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		}

		if err := downloadLocal(targetPath, downloadURL); err != nil {
			return fmt.Errorf("download locally: %w", err)
		}
	}

	// 刷新存储池以识别新文件
	if err := client.RefreshStoragePool(poolName); err != nil {
		return fmt.Errorf("refresh storage pool: %w", err)
	}

	return nil
}

// downloadLocal 在本地下载文件
func downloadLocal(targetPath, downloadURL string) error {
	// 优先使用 wget，如果不存在则使用 curl
	wgetPath, err := exec.LookPath("wget")
	if err == nil {
		cmd := exec.Command(wgetPath, "-q", "-O", targetPath, downloadURL)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("wget failed: %w, output: %s", err, string(output))
		}
		return nil
	}

	curlPath, err := exec.LookPath("curl")
	if err == nil {
		cmd := exec.Command(curlPath, "-sSL", "-o", targetPath, downloadURL)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("curl failed: %w, output: %s", err, string(output))
		}
		return nil
	}

	return fmt.Errorf("neither wget nor curl is available")
}
