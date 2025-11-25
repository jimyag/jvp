# JVP 重构实现方案

## 当前代码分析

### 现有架构

```
internal/jvp/
├── api/          # HTTP API 层
│   ├── instance.go       # 虚拟机 API
│   ├── volume.go         # 存储卷 API
│   ├── image.go          # 镜像 API
│   ├── keypair.go        # 密钥对 API
│   ├── storage_pool.go   # 存储池 API
│   ├── template.go       # 模板 API
│   └── console_ws.go     # 控制台 WebSocket
├── service/      # 业务逻辑层
│   ├── instance.go       # 虚拟机服务
│   ├── volume.go         # 存储卷服务
│   ├── image.go          # 镜像服务
│   ├── keypair.go        # 密钥对服务
│   ├── storage.go        # 存储服务
│   └── console.go        # 控制台服务
├── entity/       # 数据实体
│   ├── instance.go
│   ├── volume.go
│   ├── image.go
│   └── storage.go
└── config/       # 配置

pkg/
├── libvirt/      # Libvirt 客户端封装
├── qemuimg/      # qemu-img 工具封装
├── cloudinit/    # cloud-init 配置生成
├── virtcustomize/# virt-customize 工具封装
└── ginx/         # Gin 扩展工具

web/              # 前端（Next.js）
├── app/
│   ├── instances/
│   ├── volumes/
│   ├── images/
│   ├── storage-pools/
│   ├── templates/
│   └── keypairs/
└── components/
```

### 存在的问题

1. 模块划分不清晰
   - instance.go 包含了快照管理、密码重置等功能
   - image 和 template 概念混淆
   - 缺少独立的 snapshot 和 network 模块

2. 命名不一致
   - 有的叫 instance，有的叫 vm
   - 有的叫 image，有的叫 template

3. API 风格不统一
   - 部分使用 RESTful 风格
   - 部分使用自定义路由

4. 缺少 node 和 network 模块
   - 没有节点管理功能
   - 网络管理功能不完善

## 重构目标架构

### 模块划分（按依赖关系）

```
底层 → 中间层 → 上层

Node (节点)
  ↓
Storage Pool (存储池)
  ↓
├─ Template (模板)
├─ Volume (存储卷)
└─ Snapshot (快照)
  ↓
Network (网络)
  ↓
VM (虚拟机)
```

### 新的目录结构

```
internal/jvp/
├── api/
│   ├── node.go           # 节点 API
│   ├── storage_pool.go   # 存储池 API
│   ├── template.go       # 模板 API
│   ├── volume.go         # 存储卷 API
│   ├── snapshot.go       # 快照 API（新增）
│   ├── network.go        # 网络 API（新增）
│   └── vm.go            # 虚拟机 API（重命名）
├── service/
│   ├── node.go           # 节点服务
│   ├── storage_pool.go   # 存储池服务
│   ├── template.go       # 模板服务
│   ├── volume.go         # 存储卷服务
│   ├── snapshot.go       # 快照服务（新增）
│   ├── network.go        # 网络服务（新增）
│   └── vm.go            # 虚拟机服务（重命名）
├── entity/
│   ├── node.go
│   ├── storage_pool.go
│   ├── template.go
│   ├── volume.go
│   ├── snapshot.go       # 快照实体（新增）
│   ├── network.go        # 网络实体（新增）
│   └── vm.go
└── config/

web/
├── app/
│   ├── nodes/            # 节点管理
│   ├── storage-pools/    # 存储池管理
│   ├── templates/        # 模板管理
│   ├── volumes/          # 存储卷管理
│   ├── snapshots/        # 快照管理（新增）
│   ├── networks/         # 网络管理（新增）
│   └── vms/             # 虚拟机管理（重命名）
└── components/
```

## 实现顺序

按照依赖关系从底层到上层实现：

### 阶段 1：底层基础（第 1-2 周）

#### 1.1 Node 模块 ✅ **已完成 (2025-11-25)**
- [x] 后端 service: node.go（节点管理服务）
- [x] 后端 entity: node.go（节点实体定义）
- [x] 后端 API: node.go（节点 API 接口）
- [x] 前端页面：app/nodes/（节点管理页面）

功能：
- [x] 添加/删除节点
- [x] 列举节点
- [x] 启用/禁用节点（维护模式）
- [x] 查询节点详情（CPU、内存、NUMA、HugePages、虚拟化特性）
- [x] 查询硬件信息（PCI、GPU、USB、网络、磁盘）
- [x] 查询节点上的虚拟机列表

技术实现：
- 使用 libvirt metadata 存储节点配置（`/var/lib/jvp/metadata/nodes/<node-name>.yaml`）
- 通过 libvirt API 查询节点硬件信息（Capabilities、Sysinfo、NodeDevices）
- PCI 设备过滤识别 GPU（class code 0x03）
- NVMe 磁盘类型识别（根据设备名 `/dev/nvme*`）
- 前端使用 Modal 弹窗展示设备详情

#### 1.2 Storage Pool 模块 ✅ **已完成 (2025-11-25)**
- [x] 重构 service/storage.go → service/storage_pool.go
- [x] 重构 entity/storage.go → entity/storage_pool.go
- [x] 重构 API 路由为 Action 风格
- [x] 前端页面优化

功能：
- 创建/删除存储池
- 启动/停止存储池
- 列举/查询存储池
- 刷新存储池

技术实现：
- 纯 libvirt API 调用，不存储额外的元数据
- 使用 NodeStorage 支持本地和远程节点操作
- Action 风格 API 路由：/api/list-storage-pools, /api/create-storage-pool 等
- 移除 StoragePool.Volumes 字段，改为只存储 VolumeCount
- StoragePoolService 直接调用 libvirt.Client，不依赖 StorageService
- 更新相关服务（VolumeService, ImageService）适配新的 StoragePool 结构

前端实现：
- 使用 Header 组件统一页面头部样式
- 支持创建存储池（Modal 弹窗）
- 支持启动/停止/刷新/删除存储池（下拉菜单）
- 可展开/折叠查看存储池中的卷列表
- 实时显示存储池容量使用情况（进度条）
- 通过现有 Volume API 获取卷列表并按池名过滤

### 阶段 2：存储层（第 3-4 周）

#### 2.1 Template 模块（需拆分重构）
- [ ] 从 image.go 拆分出 template.go
- [ ] 后端 service: template.go
- [ ] 后端 entity: template.go
- [ ] 后端 API: template.go
- [ ] 前端页面：app/templates/

功能：
- 注册模板（URL/本地文件/快照）
- 列举/查询模板
- 删除模板
- 更新模板

#### 2.2 Volume 模块（已有，需重构）
- [ ] 重构 service/volume.go
- [ ] 重构 entity/volume.go
- [ ] 重构 API 路由为 Action 风格
- [ ] 前端页面优化

功能：
- 创建 Volume（空白/从模板/从快照）
- 删除 Volume
- 列举/查询 Volume
- 扩容/克隆/压缩 Volume

#### 2.3 Snapshot 模块（新增）
- [ ] 后端 service: snapshot.go（快照服务）
- [ ] 后端 entity: snapshot.go（快照实体）
- [ ] 后端 API: snapshot.go（快照 API）
- [ ] 前端页面：app/snapshots/

功能：
- 列举/查询快照
- 删除快照
- 从快照创建虚拟机
- 导出快照为模板

注意：快照的创建在 VM 模块中

### 阶段 3：网络层（第 5 周）

#### 3.1 Network 模块（新增）
- [ ] 后端 service: network.go
- [ ] 后端 entity: network.go
- [ ] 后端 API: network.go
- [ ] 前端页面：app/networks/

功能：
- 创建/删除网络
- 启动/停止网络
- 列举/查询网络
- 配置 DHCP
- 分配静态 IP
- 配置防火墙规则（可选）
- 配置端口转发（可选）

### 阶段 4：虚拟机层（第 6-7 周）

#### 4.1 VM 模块（重构）
- [ ] 重命名 instance.go → vm.go
- [ ] 重构 service/vm.go（移除快照管理）
- [ ] 重构 entity/vm.go
- [ ] 重构 API 路由为 Action 风格
- [ ] 前端页面：app/vms/

功能：
- 创建/删除虚拟机
- 启动/停止/重启虚拟机
- 列举/查询虚拟机
- 修改虚拟机配置
- 附加/分离 Volume
- 创建快照（调用 Snapshot 服务）
- 回滚快照（调用 Snapshot 服务）
- VNC/Serial Console
- 重置密码/密钥
- 重置系统

### 阶段 5：集成与优化（第 8 周）

- [ ] API 统一为 Action 风格
- [ ] 错误处理优化
- [ ] 日志记录完善
- [ ] 单元测试
- [ ] 集成测试
- [ ] 文档更新

## 技术细节

### API 风格统一

所有 API 统一使用 Action 风格：

```go
// 旧的 RESTful 风格（废弃）
POST   /api/vms
GET    /api/vms/:id
DELETE /api/vms/:id

// 新的 Action 风格
POST /api/create-vm
POST /api/start-vm
POST /api/stop-vm
POST /api/delete-vm
POST /api/list-vms
POST /api/describe-vm
```

### Service 层设计

每个 Service 负责单一职责：

```go
type NodeService struct {
    libvirtClient *libvirt.Client
}

type StoragePoolService struct {
    libvirtClient *libvirt.Client
    nodeService   *NodeService
}

type TemplateService struct {
    storagePoolService *StoragePoolService
    libvirtClient      *libvirt.Client
}

type VolumeService struct {
    storagePoolService *StoragePoolService
    templateService    *TemplateService
    libvirtClient      *libvirt.Client
}

type SnapshotService struct {
    storagePoolService *StoragePoolService
    libvirtClient      *libvirt.Client
}

type NetworkService struct {
    nodeService   *NodeService
    libvirtClient *libvirt.Client
}

type VMService struct {
    volumeService   *VolumeService
    templateService *TemplateService
    snapshotService *SnapshotService
    networkService  *NetworkService
    nodeService     *NodeService
    libvirtClient   *libvirt.Client
}
```

### 数据存储

使用 libvirt XML + 本地文件：

- 虚拟机配置：libvirt domain XML
- 存储池配置：libvirt storage pool XML
- 网络配置：libvirt network XML
- 模板数据：存储在对应的 storage pool 中的 `_templates_/` 目录
- 模板元数据：按节点和模板分目录存储
  ```
  /var/lib/jvp/metadata/
  ├── node1/
  │   ├── template1/
  │   │   └── metadata.yaml
  │   ├── template2/
  │   │   └── metadata.yaml
  │   └── ubuntu-22.04/
  │       └── metadata.yaml
  ├── node2/
  │   └── debian-12/
  │       └── metadata.yaml
  └── local/                    # 本地节点
      └── alpine-3.18/
          └── metadata.yaml
  ```
- 密钥对：本地文件（`/var/lib/jvp/keypairs/`）

#### Template 元数据结构

metadata.yaml 示例：

```yaml
# 基本信息
name: ubuntu-22.04-server
uuid: 550e8400-e29b-41d4-a716-446655440000
type: cloud                      # cloud | snapshot
created_at: "2025-11-25T10:00:00Z"
updated_at: "2025-11-25T10:00:00Z"

# 存储信息
storage:
  pool: default                  # 所属存储池
  path: /var/lib/jvp/images/_templates_/ubuntu-22.04-server.qcow2
  format: qcow2                  # qcow2 | raw
  size: 2147483648              # 字节

# 操作系统信息
os:
  name: ubuntu
  version: "22.04"
  arch: x86_64
  kernel: "5.15.0-generic"

# 来源信息（cloud image）
source:
  type: url                      # url | file | snapshot
  url: https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img
  checksum: sha256:abcdef...

# 或来源信息（snapshot）
# source:
#   type: snapshot
#   snapshot_id: snap-123456
#   vm_id: vm-789012

# 特性标记
features:
  cloud_init: true
  virtio: true
  qemu_guest_agent: false

# 使用统计
usage:
  vm_count: 5                    # 基于此模板创建的虚拟机数量
  last_used: "2025-11-25T12:00:00Z"

# 标签和描述
tags:
  - production
  - web-server
description: Ubuntu 22.04 LTS Server Cloud Image
```

#### 元数据管理

1. 创建 template 时：
   - 镜像文件下载/复制到 storage pool 的 `_templates_/` 目录
   - 在 `/var/lib/jvp/metadata/<node>/<template>/metadata.yaml` 创建元数据文件

2. 查询 template 时：
   - 扫描 `/var/lib/jvp/metadata/<node>/` 目录
   - 读取各个 template 的 metadata.yaml

3. 删除 template 时：
   - 删除 storage pool 中的镜像文件
   - 删除对应的元数据目录

4. 多节点支持：
   - 每个节点有独立的元数据目录
   - 查询时可以指定节点或查询所有节点
   - 本地节点使用 `local` 作为目录名

### 前端重构

- 统一命名：instance → vm
- 新增 snapshots、networks、nodes 页面
- 优化 UI 组件复用
- 统一 API 调用方式

## 重构策略

### 彻底重构（不保留旧 API）

1. 删除旧的 API 路由和处理函数
2. 按新架构完全重写代码
3. 前后端同步更新，统一切换
4. 提供数据迁移工具（如需要）

### 重构步骤

1. 创建新的代码结构（与旧代码并存）
2. 逐模块实现新功能
3. 完成测试后删除旧代码
4. 更新前端调用新 API

### 测试策略

1. 单元测试：每个 Service 方法
2. 集成测试：API 端到端测试
3. 手动测试：完整流程验证
4. 测试覆盖率目标：> 80%

## 时间规划

- 阶段 1（Node + Storage Pool）：2 周
- 阶段 2（Template + Volume + Snapshot）：2 周
- 阶段 3（Network）：1 周
- 阶段 4（VM）：2 周
- 阶段 5（集成与优化）：1 周

总计：8 周

## 风险与挑战

1. 数据迁移：需要处理现有 libvirt 数据到新元数据结构
2. 测试覆盖：需要完整的测试用例验证功能正确性
3. 文档更新：需要同步更新所有文档
4. 前后端协调：需要同步更新前后端代码

## 成功标准

1. 所有模块按照新架构实现
2. API 统一为 Action 风格
3. 前后端命名一致
4. 测试覆盖率 > 80%
5. 文档完整准确
