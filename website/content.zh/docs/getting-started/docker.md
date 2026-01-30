---
title: Docker 部署
weight: 2
---

# Docker 部署

Docker 部署会在容器内运行 libvirtd，完全接管宿主机的虚拟化环境。

> [!WARNING]
> **注意：** 这将停止宿主机的 libvirt 服务。如有需要，请先备份现有虚拟机。

## 步骤 1：停止宿主机 Libvirt 服务

```bash
sudo systemctl stop libvirtd libvirtd.socket virtlogd virtlogd.socket
sudo systemctl disable libvirtd libvirtd.socket virtlogd virtlogd.socket
```

## 步骤 2：创建数据目录

```bash
sudo mkdir -p /var/lib/jvp
```

## 步骤 3：启动容器

### 使用 docker-compose（推荐）

```bash
docker compose up -d
```

### 使用 docker run

```bash
docker run -d \
  --name jvp \
  --hostname jvp \
  --privileged \
  --network host \
  --cgroup host \
  --device /dev/kvm:/dev/kvm \
  --device /dev/net/tun:/dev/net/tun \
  --device /dev/vhost-net:/dev/vhost-net \
  -v /var/lib/libvirt:/var/lib/libvirt \
  -v /var/run/libvirt:/var/run/libvirt \
  -v /etc/libvirt:/etc/libvirt \
  -v /var/lib/jvp:/var/lib/jvp \
  -e TZ=Asia/Shanghai \
  -e JVP_ADDRESS=0.0.0.0:7777 \
  -e JVP_DATA_DIR=/var/lib/jvp \
  -e LIBVIRT_URI=qemu:///system \
  --restart unless-stopped \
  ghcr.io/jimyag/jvp:latest
```

## 步骤 4：访问 Web 界面

打开浏览器访问：

```
http://<服务器IP>:7777
```
