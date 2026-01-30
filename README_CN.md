# JVP

jimyag's virtualization platform

[English](README.md) | 中文

## 简介

JVP 是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，通过 RESTful API 和现代化 Web 界面提供完整的虚拟机生命周期管理。

📖 **文档**: [https://jvp.jimyag.com](https://jvp.jimyag.com)

![实例列表](docs/static/instance.png)

## 功能特性

- **实例管理** - 创建、启动、停止、快照虚拟机，支持 cloud-init
- **多节点支持** - 管理多个 libvirt 节点（本地和远程）
- **存储管理** - 管理存储池和存储卷
- **快照与模板** - 创建快照，注册和管理虚拟机模板
- **现代化 Web 界面** - 基于 React 的界面，内置 VNC 和串口控制台

## 快速开始

### Docker（推荐）

```bash
# 停止宿主机 libvirt 服务
sudo systemctl stop libvirtd libvirtd.socket virtlogd virtlogd.socket
sudo systemctl disable libvirtd libvirtd.socket virtlogd virtlogd.socket

# 创建数据目录
sudo mkdir -p /var/lib/jvp

# 启动容器
docker compose up -d
```

访问: `http://<服务器IP>:7777`

### 二进制文件

从 [GitHub Releases](https://github.com/jimyag/jvp/releases) 下载后运行:

```bash
./jvp
```

## 文档

详细的安装指南、功能文档和 API 参考，请访问:

**[https://jvp.jimyag.com](https://jvp.jimyag.com)**

## 许可证

[MIT](LICENSE)
