# JVP

jimyag's virtualization platform

English | [中文](README_CN.md)

## Introduction

JVP is a virtualization platform based on QEMU/KVM and libvirt, providing complete virtual machine lifecycle management through RESTful API and a modern web interface.

📖 **Documentation**: [https://jvp.jimyag.com](https://jvp.jimyag.com)

![Instance List](docs/static/instance.png)

## Features

- **Instance Management** - Create, start, stop, snapshot VMs with cloud-init support
- **Multi-Node Support** - Manage multiple libvirt nodes (local and remote)
- **Storage Management** - Manage storage pools and volumes
- **Snapshot & Template** - Create snapshots, register and manage VM templates
- **Modern Web UI** - React-based interface with VNC and Serial console

## Quick Start

### Docker (Recommended)

```bash
# Stop host libvirt services
sudo systemctl stop libvirtd libvirtd.socket virtlogd virtlogd.socket
sudo systemctl disable libvirtd libvirtd.socket virtlogd virtlogd.socket

# Create data directory
sudo mkdir -p /var/lib/jvp

# Start container
docker compose up -d
```

Access: `http://<server-ip>:7777`

### Binary

Download from [GitHub Releases](https://github.com/jimyag/jvp/releases) and run:

```bash
./jvp
```

## Documentation

For detailed installation guides, feature documentation, and API reference, visit:

**[https://jvp.jimyag.com](https://jvp.jimyag.com)**

## License

[MIT](LICENSE)
