---
title: 实例管理
weight: 1
---

# 实例管理

JVP 提供完整的虚拟机生命周期管理。

## 创建实例

- 自定义 CPU、内存和磁盘
- 支持桥接或 NAT 网络
- 集成 cloud-init，支持用户数据和 SSH 公钥注入

## 查询实例

- 按节点或 ID 查询
- 返回网络接口、MAC、IP 地址
- 显示自启动标志和启动时间

## 生命周期管理

- **启动** - 启动虚拟机
- **停止** - 优雅关机或强制停止
- **重启** - 重启虚拟机
- **删除** - 删除实例（可选删除卷）

## 修改实例属性

- 调整 CPU 和内存
- 更改实例名称
- 配置自启动行为

## 密码重置

- 基于 guest-agent 的异步重置
- 后台执行，支持 virt-customize 备选方案

## 远程控制台

- **VNC 控制台** - 图形化远程访问
- **串口控制台** - 文本控制台访问

![实例详情](/images/instance-detail.png)

![VNC 控制台](/images/instance-vnc.png)

![串口控制台](/images/instance-console.png)
