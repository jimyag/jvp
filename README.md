# JVP

jimyag’s virtualization platform. jimyag 的虚拟化平台

## 架构介绍

JVP 是一个基于 qemu 的虚拟化平台，拥有以下组件：

**jvp** 为一个命令行虚拟机管理的客户端工具，提供了虚拟机的启动和停止，虚拟机的配置管理等功能。

**jvp-daemon** 是 JVP 的守护进程，负责物理机上管理虚拟机、上报虚拟机的状态等。需要在每台物理机上安装。

**jvp-hub** 是 JVP 的中心枢纽，对外提供 API 服务、虚拟机的调度管理、下发任务给 jvp-daemon 执行。
可以在任何一个节点部署，默认部署一台即可，为了高可用可以在多个节点部署，
他们会选举出一个 leader 对外提供服务，如果 leader 宕机，会自动选举出下一个节点作为 leader。

**jvp-admin** 是 JVP 的管理工具，提供一个 Web 界面，可以方便的管理虚拟机。

## 相关资料

- <https://www.voidking.com/dev-libvirt-create-vm/>
- <https://sq.sf.163.com/blog/article/172808502565068800>
- <https://shihai1991.github.io/openstack/2024/02/20/%E9%80%9A%E8%BF%87libvirt%E5%88%9B%E5%BB%BA%E8%99%9A%E6%8B%9F%E6%9C%BA/>
- <https://www.baeldung.com/linux/qemu-uefi-boot> 启动 qemu 的 UEFI 引导
