---
title: 节点管理
weight: 2
---

# 节点管理

JVP 支持管理多个 libvirt 节点，构建分布式虚拟化集群。

## 节点类型

- **本地节点** - 自动创建 `local (qemu:///system)` 节点
- **远程节点** - 通过 libvirt URI 添加（如 `qemu+ssh://user@host/system`）
- **节点类型** - 计算、存储、混合等类型

## 节点操作

- 添加新节点
- 删除现有节点
- 启用/禁用节点
- 查看节点摘要

## 节点摘要

查看每个节点的硬件信息：

- CPU 信息
- 内存容量
- NUMA 拓扑
- 虚拟化能力

![节点管理](/images/nodes.png)
