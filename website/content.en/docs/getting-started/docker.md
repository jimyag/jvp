---
title: Docker Deployment
weight: 2
---

# Docker Deployment

Docker deployment runs libvirtd inside the container, completely taking over the host's virtualization environment.

{{< hint warning >}}
**Note:** This will stop the host's libvirt services. Make sure to backup any existing VMs if needed.
{{< /hint >}}

## Step 1: Stop Host Libvirt Services

```bash
sudo systemctl stop libvirtd libvirtd.socket virtlogd virtlogd.socket
sudo systemctl disable libvirtd libvirtd.socket virtlogd virtlogd.socket
```

## Step 2: Create Data Directory

```bash
sudo mkdir -p /var/lib/jvp
```

## Step 3: Start the Container

### Using docker-compose (Recommended)

```bash
docker compose up -d
```

### Using docker run

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

## Step 4: Access Web Interface

Open your browser and navigate to:

```
http://<server-ip>:7777
```
