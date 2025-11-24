# 模块

## instance

包含 实例的生命周期的管理

xml 定义操作是什么？

查询运行状态 cpu/mem/disk/network等 IO 指标

## snapshot
从实例导出为快照
快照的创建、删除、回滚、导入为实例

## storage pool

不同的类型

- dir 目录
- fs 挂载点
- lvm
- zfs 
- rbd

对应的操作 
1. 创建存储池
2. 删除存储池
3. 添加卷
4. 删除卷
5. 挂载卷
6. 卸载卷
7. 获取卷的信息
8. 获取存储池的信息
9. 获取卷列表
10 获取存储池列表

## template

1. cloud-template 比如 ubuntu、debian、alpine的官方的 cloud image 
2. 从某个 snapshot 导出为 template 



