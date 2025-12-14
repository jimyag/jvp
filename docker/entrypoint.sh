#!/bin/bash
set -e

echo "=== JVP Container Starting ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"

# 检查 KVM 支持
echo ""
echo "=== Checking KVM support ==="
if [ -e /dev/kvm ]; then
    echo "[OK] /dev/kvm exists"
    # 确保权限正确
    chmod 666 /dev/kvm 2>/dev/null || true
else
    echo "[WARN] /dev/kvm not found - VMs will run without KVM acceleration (slow)"
fi

# 检查 TUN 设备
if [ -e /dev/net/tun ]; then
    echo "[OK] /dev/net/tun exists"
else
    echo "[WARN] /dev/net/tun not found - network may not work"
    # 尝试创建
    mkdir -p /dev/net
    mknod /dev/net/tun c 10 200 2>/dev/null || true
    chmod 666 /dev/net/tun 2>/dev/null || true
fi

# 创建必要的目录
echo ""
echo "=== Creating directories ==="
mkdir -p /var/run/libvirt
mkdir -p /var/lib/libvirt/images
mkdir -p /var/lib/libvirt/qemu
mkdir -p /var/log/libvirt/qemu
mkdir -p /app/data

# 启动 virtlogd（libvirt 日志守护进程）
echo ""
echo "=== Starting virtlogd ==="
if command -v virtlogd &> /dev/null; then
    virtlogd -d || echo "[WARN] Failed to start virtlogd"
fi

# 启动 libvirtd
echo ""
echo "=== Starting libvirtd ==="
libvirtd -d -l
sleep 2

# 检查 libvirtd 状态
if pgrep -x libvirtd > /dev/null; then
    echo "[OK] libvirtd is running"
else
    echo "[ERROR] libvirtd failed to start"
    exit 1
fi

# 配置默认网络
echo ""
echo "=== Configuring default network ==="
# 检查默认网络是否已定义
if ! virsh net-info default &>/dev/null; then
    echo "Defining default network..."
    virsh net-define /etc/libvirt/qemu/networks/default.xml || true
fi

# 启动默认网络
if virsh net-info default 2>/dev/null | grep -q "Active:.*no"; then
    echo "Starting default network..."
    virsh net-start default || true
fi

# 设置默认网络自动启动
virsh net-autostart default 2>/dev/null || true

# 显示网络状态
echo ""
echo "=== Network status ==="
virsh net-list --all

# 显示存储池状态
echo ""
echo "=== Storage pools ==="
virsh pool-list --all

# 检查连接
echo ""
echo "=== Testing libvirt connection ==="
if virsh -c qemu:///system version; then
    echo "[OK] libvirt connection successful"
else
    echo "[ERROR] Cannot connect to libvirt"
    exit 1
fi

# 启动 supervisord（管理 JVP 进程）
echo ""
echo "=== Starting supervisord ==="
exec /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf
