# 单元测试补充任务

## 已完成测试

### VolumeService
- [x] TestVolumeService_DescribeEBSVolumes_Pagination
- [x] TestVolumeService_DeleteEBSVolume
- [x] TestVolumeService_AttachEBSVolume
- [x] TestVolumeService_DetachEBSVolume
- [x] TestVolumeService_DescribeEBSVolume
- [x] TestVolumeService_ModifyEBSVolume
- [x] TestNewVolumeService

## 测试覆盖率

当前测试覆盖率：**67.8%**

### 并发问题修复

已修复所有并发相关的测试失败：
- 为每个测试用例创建独立的 mock client，避免共享 mock 导致的冲突
- 移除了可能导致并发冲突的 `t.Parallel()` 调用（在子测试中）
- 每个测试用例使用独立的 service 实例

## 已完成测试（全部）

### ImageService
- [x] TestImageService_RegisterImage
- [x] TestImageService_GetImage
- [x] TestImageService_DescribeImages
- [x] TestImageService_ListImages
- [x] TestImageService_DeleteImage
- [x] TestImageService_CreateImageFromInstance
- [x] TestImageService_DownloadImage
- [x] TestImageService_EnsureDefaultImages
- [x] TestImageService_GetDefaultImageByName
- [x] TestImageService_ListDefaultImages
- [x] TestNewImageService

### InstanceService
- [x] TestInstanceService_RunInstance
- [x] TestInstanceService_DescribeInstances
- [x] TestInstanceService_GetInstance
- [x] TestInstanceService_TerminateInstances
- [x] TestInstanceService_StopInstances
- [x] TestInstanceService_StartInstances
- [x] TestInstanceService_RebootInstances
- [x] TestInstanceService_ModifyInstanceAttribute
- [x] TestNewInstanceService

### StorageService
- [x] TestStorageService_EnsurePool
- [x] TestStorageService_GetPool
- [x] TestStorageService_CreateVolume
- [x] TestStorageService_CreateVolumeFromImage
- [x] TestStorageService_DeleteVolume
- [x] TestStorageService_GetVolume
- [x] TestStorageService_ListVolumes

### Service
- [x] TestService_Run
- [x] TestService_Shutdown
- [x] TestService_New

## 测试规范

1. 使用 t.Parallel() 并行执行测试
2. 使用 testcase 结构体组织测试用例
3. 使用 testify 的 assert 函数进行断言
4. 使用 t.Run 组织子测试用例
5. 尽量复用测试代码，包含创建的各种 mock client、service
6. 尽可能 mock 依赖的组件，如果有现成的，使用现成的
7. 尽可能覆盖更多的场景
8. 测试覆盖率要高，尽量覆盖所有代码分支

