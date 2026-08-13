# JVP

jimyag's virtualization platform

[English](README.md) | 中文

## 简介

JVP 是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，通过 RESTful API 和现代化 Web 界面提供完整的 Linux 与 Windows 虚拟机生命周期管理。

📖 **文档**: [https://jvp.jimyag.com](https://jvp.jimyag.com)

![实例列表](docs/static/instance.png)

## 功能特性

- **实例管理** - 创建、启动、停止、快照虚拟机，支持 cloud-init
- **Windows 支持** - 使用 Windows ISO 和 VirtIO 驱动安装系统，或使用 Cloudbase-Init 批量创建 Windows 云镜像实例
- **多节点支持** - 管理多个 libvirt 节点（本地和远程）
- **存储管理** - 管理存储池和存储卷
- **快照与模板** - 创建快照，注册和管理虚拟机模板
- **现代化 Web 界面** - 基于 React 的界面，内置 VNC 和串口控制台

## Windows 虚拟机

JVP 支持两种 Windows 创建方式：

- **Install ISO**：创建空白系统盘，从 Windows 安装 ISO 启动，并可同时挂载 VirtIO 驱动 ISO。
- **Cloud Image**：克隆已经准备好的 Windows 磁盘模板，通过 Cloudbase-Init 和 NoCloud 配置盘完成首次启动初始化。

Cloud Image 模式可在实例创建页面指定主机名、时区、管理员用户名和密码、SSH 公钥及首次启动命令。模板需要预装 Cloudbase-Init、VirtIO 驱动和 QEMU Guest Agent，并在注册时启用 `Cloud-init / Cloudbase-Init ready` 与 `VirtIO` 特性。

![Windows 实例](docs/static/windows-instance.png)

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
