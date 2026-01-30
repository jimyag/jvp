---
title: Installation
weight: 1
---

# Installation

JVP can be deployed in several ways. Choose the method that best fits your environment.

## Requirements

### Required Tools

| Tool | Purpose |
|------|---------|
| **libvirt** | Virtualization management core (libvirtd daemon) |
| **virsh** | Execute qemu-agent-command |
| **qemu-img** | Disk image operations |
| **genisoimage** or **mkisofs** | Generate cloud-init ISO |
| **ssh** | Remote node connection |

### Optional Tools

| Tool | Purpose |
|------|---------|
| **wget** | Download template images (preferred) |
| **curl** | Download template images (fallback) |
| **ip** | Query ARP neighbor table for VM IP |
| **virt-customize** | Reset VM password (fallback method) |
| **socat** | VNC/serial forwarding for remote nodes |

### Install on Debian/Ubuntu

```bash
apt install libvirt-daemon-system qemu-utils genisoimage wget curl openssh-client libguestfs-tools socat
```

### Install on RHEL/CentOS/Fedora

```bash
dnf install libvirt qemu-img genisoimage wget curl openssh-clients libguestfs-tools socat
```

## Deployment Options

- [Docker Deployment]({{< relref "docker" >}}) (Recommended)
- [Binary Deployment]({{< relref "binary" >}})
- [Build from Source]({{< relref "source" >}})
