---
title: Node Management
weight: 2
---

# Node Management

JVP supports managing multiple libvirt nodes to build distributed virtualization clusters.

## Node Types

- **Local Node** - Automatically creates `local (qemu:///system)` node
- **Remote Nodes** - Add via libvirt URI (e.g., `qemu+ssh://user@host/system`)
- **Node Types** - Compute, storage, hybrid, and other types

## Node Operations

- Add new nodes
- Delete existing nodes
- Enable/disable nodes
- View node summary

## Node Summary

View hardware information for each node:

- CPU information
- Memory capacity
- NUMA topology
- Virtualization capabilities

![Node Management](/images/nodes.png)
