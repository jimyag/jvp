# JVP

jimyag's virtualization platform. jimyag 的虚拟化平台

## 简介

JVP 是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，提供完整的虚拟机生命周期管理功能。支持通过 RESTful API 创建、管理和监控虚拟机实例。

![JVP](docs/static/Snipaste_2025-11-29_19-36-26.png)

![JVP](docs/static/Snipaste_2025-11-29_19-37-42.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-00.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-12.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-25.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-36.png)

![JVP](docs/static/Snipaste_2025-11-29_19-38-58.png)

## 核心功能

### 实例管理（Instances）

- **创建实例**：自定义 CPU、内存、磁盘，支持桥接或 NAT 网络
  - 集成 cloud-init，支持用户数据与 SSH 公钥注入
- **查询实例**：按节点/ID 查询，返回网卡、MAC、IP、多 IP 信息、开机自启动标记、启动时间
- **生命周期管理**：启动、停止、重启、删除（可选同时删除卷）
- **修改实例属性**：调整 CPU、内存、名称、自动启动
- **密码重置**：基于 guest-agent 的异步重置（后台执行），保留 virt-customize 兜底

### 节点与存储

- **节点管理**：支持本地/远程 libvirt；无配置时自动创建 `local (qemu:///system)` 节点
- **节点概要**：CPU/内存/NUMA/虚拟化能力概览
- **存储池/卷**：列举/创建/启停/删除存储池，列举卷，创建卷并可随实例删除

### 密钥对管理（KeyPairs）

- **创建密钥对**：支持 RSA 和 ED25519 算法
- **导入密钥对**：导入现有公钥
- **查询密钥对**：支持按 ID、名称等条件查询
- **删除密钥对**：删除不再使用的密钥对
- **自动注入**：创建实例时自动注入 SSH 公钥

## 如何使用

```bash
task build

./bin/jvp
```

## 相关资料

- <https://www.voidking.com/dev-libvirt-create-vm/>
- <https://sq.sf.163.com/blog/article/172808502565068800>
- <https://shihai1991.github.io/openstack/2024/02/20/%E9%80%9A%E8%BF%87libvirt%E5%88%9B%E5%BB%BA%E8%99%9A%E6%8B%9F%E6%9C%BA/>
- <https://www.baeldung.com/linux/qemu-uefi-boot> 启动 qemu 的 UEFI 引导
