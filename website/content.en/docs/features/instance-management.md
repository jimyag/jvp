---
title: Instance Management
weight: 1
---

# Instance Management

JVP provides complete virtual machine lifecycle management.

## Create Instances

- Customize CPU, memory, and disk
- Support bridge or NAT networking
- Integrated cloud-init with user data and SSH public key injection

## Query Instances

- Query by node or ID
- Returns network interfaces, MAC, IP addresses
- Shows autostart flag and start time

## Lifecycle Management

- **Start** - Boot the virtual machine
- **Stop** - Gracefully shutdown or force stop
- **Reboot** - Restart the virtual machine
- **Delete** - Remove instance (optionally delete volumes)

## Modify Instance Properties

- Adjust CPU and memory
- Change instance name
- Configure autostart behavior

## Password Reset

- Asynchronous reset based on guest-agent
- Background execution with virt-customize fallback

## Remote Console

- **VNC Console** - Graphical remote access
- **Serial Console** - Text-based console access

![Instance Details](/images/instance-detail.png)

![VNC Console](/images/instance-vnc.png)

![Serial Console](/images/instance-console.png)
