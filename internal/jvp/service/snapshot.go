package service

// EBS Snapshot 功能已移除
// 
// JVP 不再支持 EBS 快照（卷级别快照）。
// 请使用 libvirt Domain Snapshots（虚拟机级别快照）替代。
//
// Domain Snapshots 的优势：
// 1. 快照整个虚拟机状态（包括内存、磁盘）
// 2. libvirt 原生支持，无需额外存储
// 3. 可以直接从快照克隆新虚拟机
//
// 如何使用 Domain Snapshots：
// 
// 创建快照：
//   virsh snapshot-create-as <domain-name> <snapshot-name> --description "描述"
//
// 列出快照：
//   virsh snapshot-list <domain-name>
//
// 恢复快照：
//   virsh snapshot-revert <domain-name> <snapshot-name>
//
// 删除快照：
//   virsh snapshot-delete <domain-name> <snapshot-name>
//
// VM Templates 功能会自动列出所有带快照的虚拟机，
// 可以作为模板使用。
