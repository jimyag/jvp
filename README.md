# JVP

jimyag's virtualization platform. jimyag 的虚拟化平台

[English](README_EN.md) | 中文

## 简介

JVP 是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，提供完整的虚拟机生命周期管理功能。支持通过 RESTful API 和现代化的 Web 管理界面创建、管理和监控虚拟机实例。

![JVP](docs/static/Snipaste_2025-11-29_19-36-26.png)

![JVP](docs/static/Snipaste_2025-11-29_19-37-42.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-00.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-12.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-25.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-36.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-58.png)

![JVP](docs/static/Snipaste_2025-12-02_00-00-03.png)

![JVP](docs/static/Snipaste_2025-12-02_00-00-16.png)

## 核心功能

### 实例管理（Instances）

- **创建实例**：自定义 CPU、内存、磁盘，支持桥接或 NAT 网络
  - 集成 cloud-init，支持用户数据与 SSH 公钥注入
- **查询实例**：按节点/ID 查询，返回网卡、MAC、IP、多 IP 信息、开机自启动标记、启动时间
- **生命周期管理**：启动、停止、重启、删除（可选同时删除卷）
- **修改实例属性**：调整 CPU、内存、名称、自动启动
- **密码重置**：基于 guest-agent 的异步重置（后台执行），保留 virt-customize 兜底
- **远程控制台**：支持 VNC/Serial 控制台

### 节点与存储

- **节点管理**：支持本地/远程 libvirt；无配置时自动创建 `local (qemu:///system)` 节点
- **节点概要**：CPU/内存/NUMA/虚拟化能力概览
- **存储池管理**：列举/创建/启停/删除存储池，查看存储池使用情况
- **存储卷管理**：列举卷，创建卷，查看卷详情（容量、分配、格式），删除卷

### 快照管理（Snapshots）

- **创建快照**：为虚拟机创建快照，支持包含内存状态
- **列举快照**：按节点和虚拟机查询快照列表
- **快照详情**：查看快照的详细信息（创建时间、状态、磁盘信息等）
- **回滚快照**：将虚拟机恢复到指定快照状态
- **删除快照**：删除不再需要的快照，释放存储空间
- **快照类型**：支持内部快照（qcow2）和外部快照（所有格式）

### 模板管理（Templates）

- **注册模板**：从 URL 下载云镜像或从本地文件导入
- **列举模板**：查看所有可用的虚拟机模板
- **模板详情**：查看模板的元数据（操作系统、大小、来源等）
- **删除模板**：删除不再使用的模板
- **模板类型**：支持云镜像模板（Ubuntu、Debian、Alpine 等）和快照导出的自定义模板

### 密钥对管理（KeyPairs）

- **创建密钥对**：支持 RSA 和 ED25519 算法
- **导入密钥对**：导入现有公钥
- **查询密钥对**：支持按 ID、名称等条件查询
- **删除密钥对**：删除不再使用的密钥对
- **自动注入**：创建实例时自动注入 SSH 公钥

### Web 管理界面

- **现代化 UI**：基于 React + Vite + Tailwind CSS 构建的响应式 Web 界面
- **实时监控**：查看实例状态、资源使用情况、网络信息等
- **远程控制台**：集成 VNC 和 Serial 控制台，支持图形和文本界面访问
- **统一管理**：通过 Web 界面管理所有资源（实例、节点、存储、模板、快照、密钥对）

## 如何使用

### 构建项目

```bash
# 构建包含前端的完整二进制文件
task build
```

### 运行服务

```bash
# 运行 JVP 服务（默认端口 7777）
./bin/jvp
```

### 访问 Web 界面

构建完成后，前端已嵌入到二进制文件中。启动服务后访问：

```bash
http://localhost:7777
```

## 相关资料

- <https://www.voidking.com/dev-libvirt-create-vm/>
- <https://sq.sf.163.com/blog/article/172808502565068800>
- <https://shihai1991.github.io/openstack/2024/02/20/%E9%80%9A%E8%BF%87libvirt%E5%88%9B%E5%BB%BA%E8%99%9A%E6%8B%9F%E6%9C%BA/>
- <https://www.baeldung.com/linux/qemu-uefi-boot> 启动 qemu 的 UEFI 引导
