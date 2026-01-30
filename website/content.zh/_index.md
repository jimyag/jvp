---
title: JVP
type: docs
---

# JVP - 虚拟化平台

**JVP**（jimyag's virtualization platform）是一个基于 QEMU/KVM 和 libvirt 的虚拟化平台，提供完整的虚拟机生命周期管理。

{{% columns %}}
- ### 简单易用
  使用 Docker 几分钟即可部署，无需任何配置即可开始使用。

  ```bash
  docker compose up -d
  ```

- ### 功能强大
  完整的虚拟机生命周期管理：创建、启动、停止、快照、模板等。

- ### 界面现代
  使用 React 构建的精美 Web 界面，内置 VNC 和串口控制台。
{{% /columns %}}

## 快速开始

{{< button href="/zh/docs/getting-started/installation/" >}}开始使用{{< /button >}}

## 截图展示

![实例列表](/images/instance.png)

![VNC 控制台](/images/instance-vnc.png)
