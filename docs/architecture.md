# JVP 架构设计

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    API Layer (Handler)                   │
│  - 接收 HTTP 请求                                        │
│  - 参数验证和绑定                                        │
│  - 调用 Service 层                                       │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                 Service Layer                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Instance   │  │   Storage    │  │    Image     │ │
│  │   Service    │  │   Service    │  │   Service    │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         │                 │                  │         │
│         └─────────────────┼──────────────────┘         │
│                           │                             │
└───────────────────────────┼─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│              Repository Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Instance   │  │    Volume    │  │    Image     │ │
│  │  Repository  │  │  Repository  │  │  Repository  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│            Infrastructure Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Libvirt    │  │  PostgreSQL  │  │   File       │ │
│  │   Client     │  │   Database   │  │   System     │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└──────────────────────────────────────────────────────────┘
```

## 服务职责划分

### 1. Storage Service
**职责：管理 libvirt storage pool 和 volume（磁盘）**

- **Pool 管理**
  - 创建/删除/查询 storage pool
  - 确保 pool 存在和可用
  - 管理 pool 的容量和配额

- **Volume 管理**
  - 在指定 pool 中创建 volume（磁盘文件）
  - 从镜像（template）克隆 volume
  - 删除 volume
  - 查询 volume 信息（大小、格式等）
  - 管理 volume 的生命周期

- **镜像管理（可选，也可以单独 Image Service）**
  - 上传/下载镜像到 pool
  - 从 instance 创建镜像（snapshot）
  - 管理镜像模板

### 2. Instance Service
**职责：管理虚拟机实例的生命周期**

- **Instance 创建流程**
  1. 验证请求参数
  2. 调用 Storage Service 创建磁盘（从镜像或新建）
  3. 调用 libvirt Client 创建 domain
  4. 保存 instance 元数据到 Repository
  5. 返回 instance 信息

- **Instance 管理**
  - 启动/停止/重启 instance
  - 删除 instance（同时清理磁盘）
  - 查询 instance 状态
  - 管理 instance 配置（CPU、内存等）

### 3. Image Service（可选）
**职责：管理镜像模板**

- 上传镜像到 storage pool
- 从 instance 创建镜像
- 管理镜像元数据
- 镜像版本管理

## 数据流示例：创建 Instance

```
1. API Handler 接收 RunInstances 请求
   ↓
2. Instance Service.RunInstances()
   ├─ 验证参数（ImageId, InstanceType 等）
   ├─ 调用 Storage Service.CreateVolume()
   │  ├─ 检查/创建 storage pool
   │  ├─ 从 ImageId 对应的镜像克隆 volume
   │  └─ 返回 volume 路径
   ├─ 调用 libvirt Client.CreateVM()
   │  ├─ 使用 volume 路径创建 domain
   │  └─ 返回 domain
   ├─ 保存 instance 元数据到 Repository
   └─ 返回 instance 信息
```

## 关键设计决策

### 1. Storage Pool 管理策略

**方案 A：每个 Instance 使用独立的 Pool**
- 优点：隔离性好，便于管理
- 缺点：资源浪费，管理复杂

**方案 B：使用共享 Pool（推荐）**
- 优点：资源利用率高，管理简单
- 缺点：需要容量管理

**建议：使用共享 Pool，按类型划分**
- `default` pool：存储用户创建的 instance 磁盘
- `images` pool：存储镜像模板
- `snapshots` pool：存储快照

### 2. Volume 命名策略

参考 AWS EBS Volume：
- Volume ID: `vol-{uuid}`
- 磁盘文件路径：`{pool_path}/vol-{uuid}.qcow2`

### 3. Instance 与 Volume 的关系

- 一个 Instance 可以有多个 Volume（根磁盘 + 数据磁盘）
- Instance 删除时，可以选择是否删除关联的 Volume
- Volume 可以独立于 Instance 存在（类似 EBS）

### 4. 镜像管理

- 镜像存储在 `images` pool 中
- 镜像 ID: `ami-{uuid}`
- 创建 Instance 时，从镜像克隆到 `default` pool

## 目录结构建议

```
internal/jvp/
├── api/              # API Handler 层
│   ├── instance.go
│   └── volume.go
├── service/          # Service 层
│   ├── instance.go   # Instance Service
│   ├── storage.go    # Storage Service
│   └── image.go      # Image Service (可选)
├── repository/       # Repository 层
│   ├── instance.go
│   ├── volume.go
│   └── image.go
└── entity/           # 实体定义
    ├── instance.go
    ├── volume.go
    └── image.go
```

