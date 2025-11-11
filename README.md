# JVP

jimyag's virtualization platform. jimyag 的虚拟化平台

## 简介

JVP 是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，提供完整的虚拟机生命周期管理功能。支持通过 RESTful API 创建、管理和监控虚拟机实例。

## 核心功能

### 实例管理（Instances）

- **创建实例**：支持自定义 CPU、内存、磁盘等配置
  - 集成 cloud-init，支持用户数据和自定义配置
  - 支持 SSH 密钥对注入
  - 自动下载和管理系统镜像
- **查询实例**：支持按 ID、状态等条件查询
- **生命周期管理**：启动、停止、重启、终止实例
- **修改实例属性**：动态调整 CPU 和内存（需要实例支持）
- **重置密码**：支持三种策略，按优先级自动选择
  - qemu-guest-agent（优先，无需停止实例，实时生效）
  - cloud-init（需要重启实例）
  - virt-customize（最后选择，需要停止实例）

### 卷管理（Volumes）

- **创建卷**：支持从镜像或快照创建
- **查询卷**：支持按 ID、状态等条件查询
- **删除卷**：安全删除卷资源
- **附加/分离卷**：动态挂载和卸载卷到实例
- **修改卷**：支持扩容等操作

### 快照管理（Snapshots）

- **创建快照**：为卷创建时间点快照
- **查询快照**：支持按 ID、状态等条件查询
- **删除快照**：删除不再需要的快照
- **复制快照**：跨区域或跨存储池复制快照

### 镜像管理（Images）

- **创建镜像**：从运行中的实例创建镜像
- **查询镜像**：支持按 ID、名称等条件查询
- **注册镜像**：注册外部镜像到系统
- **注销镜像**：从系统中移除镜像
- **自动管理**：自动下载和管理默认系统镜像（如 Ubuntu）

### 密钥对管理（KeyPairs）

- **创建密钥对**：支持 RSA 和 ED25519 算法
- **导入密钥对**：导入现有公钥
- **查询密钥对**：支持按 ID、名称等条件查询
- **删除密钥对**：删除不再使用的密钥对
- **自动注入**：创建实例时自动注入 SSH 公钥

## 技术特性

- **纯 Go 实现**：不依赖 CGO，使用 `modernc.org/sqlite` 纯 Go SQLite 驱动
- **接口化设计**：服务层和 API 层使用接口，便于测试和扩展
- **完整的测试覆盖**：支持并发测试，包含 race 检测
- **结构化日志**：使用 zerolog 记录结构化日志
- **优雅关闭**：支持优雅关闭和资源清理

## Web 管理界面

JVP 提供了现代化的 Web 管理界面,位于 `web/` 目录。参考 MotherDuck 设计系统,提供响应式布局和简洁现代的界面。

快速启动:

```bash
cd web && ./start.sh
```

访问 `http://localhost:3000` 使用图形化界面管理虚拟化资源。详细文档: [web/README.md](web/README.md)、[web/USAGE.md](web/USAGE.md)

## API 端点

所有 API 端点位于 `/api` 路径下:

- `/api/instances/*` - 实例管理
- `/api/volumes/*` - 卷管理
- `/api/snapshots/*` - 快照管理
- `/api/images/*` - 镜像管理
- `/api/keypairs/*` - 密钥对管理

## 架构介绍

JVP 是一个基于 qemu 的虚拟化平台，支持创建和管理虚拟机、轻量容器 firecracker。

数据存储在 SQLite 数据库中（使用纯 Go 驱动，无需 CGO）。

## 相关资料

- <https://www.voidking.com/dev-libvirt-create-vm/>
- <https://sq.sf.163.com/blog/article/172808502565068800>
- <https://shihai1991.github.io/openstack/2024/02/20/%E9%80%9A%E8%BF%87libvirt%E5%88%9B%E5%BB%BA%E8%99%9A%E6%8B%9F%E6%9C%BA/>
- <https://www.baeldung.com/linux/qemu-uefi-boot> 启动 qemu 的 UEFI 引导
