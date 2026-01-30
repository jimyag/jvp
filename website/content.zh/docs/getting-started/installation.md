---
title: 安装指南
weight: 1
---

# 安装指南

JVP 支持多种部署方式，请选择最适合你环境的方式。

## 系统要求

### 必需工具

| 工具 | 用途 |
|------|------|
| **libvirt** | 虚拟化管理核心（libvirtd 守护进程） |
| **virsh** | 执行 qemu-agent-command |
| **qemu-img** | 磁盘镜像操作 |
| **genisoimage** 或 **mkisofs** | 生成 cloud-init ISO |
| **ssh** | 远程节点连接 |

### 可选工具

| 工具 | 用途 |
|------|------|
| **wget** | 下载模板镜像（首选） |
| **curl** | 下载模板镜像（备选） |
| **ip** | 查询 ARP 邻居表获取虚拟机 IP |
| **virt-customize** | 重置虚拟机密码（备选方法） |
| **socat** | 远程节点的 VNC/串口转发 |

### 在 Debian/Ubuntu 上安装

```bash
apt install libvirt-daemon-system qemu-utils genisoimage wget curl openssh-client libguestfs-tools socat
```

### 在 RHEL/CentOS/Fedora 上安装

```bash
dnf install libvirt qemu-img genisoimage wget curl openssh-clients libguestfs-tools socat
```

## 部署方式

- [Docker 部署]({{< relref "docker" >}})（推荐）
- [二进制部署]({{< relref "binary" >}})
- [源码编译]({{< relref "source" >}})
