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
    chmod 666 /dev/kvm 2>/dev/null || true
else
    echo "[WARN] /dev/kvm not found - VMs will run without KVM acceleration (slow)"
fi

# 检查 TUN 设备
if [ -e /dev/net/tun ]; then
    echo "[OK] /dev/net/tun exists"
else
    echo "[WARN] /dev/net/tun not found - network may not work"
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
mkdir -p /var/log/supervisor
mkdir -p /app/data
mkdir -p /var/lib/jvp/qemu
chown libvirt-qemu:kvm /var/lib/jvp/qemu 2>/dev/null || true

# 清理残留的 socket 和 PID 文件（从宿主机挂载可能残留）
echo ""
echo "=== Cleaning up stale files ==="
rm -f /var/run/libvirt/libvirt-sock*
rm -f /var/run/libvirt/virtlogd-sock
rm -f /var/run/libvirt/virtlockd-sock
rm -f /var/run/libvirtd.pid
rm -f /var/run/virtlogd.pid

# 启动 virtlogd（libvirt 日志守护进程）
echo ""
echo "=== Starting virtlogd ==="
if command -v virtlogd &> /dev/null; then
    virtlogd -d || echo "[WARN] Failed to start virtlogd"
fi

# 启动 libvirtd（不使用 -l，只监听本地 socket）
echo ""
echo "=== Starting libvirtd ==="
libvirtd -d

# 等待 libvirtd 就绪（带重试）
echo "Waiting for libvirtd to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! virsh -c qemu:///system version &>/dev/null; do
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "[ERROR] libvirtd failed to start after ${MAX_RETRIES} seconds"
        exit 1
    fi
    echo "  Waiting... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 1
done
echo "[OK] libvirtd is ready"

# 配置默认网络
echo ""
echo "=== Configuring default network ==="
if ! virsh net-info default &>/dev/null; then
    echo "Defining default network..."
    virsh net-define /etc/libvirt/qemu/networks/default.xml || true
fi

if virsh net-info default 2>/dev/null | grep -q "Active:.*no"; then
    echo "Starting default network..."
    virsh net-start default || true
fi

virsh net-autostart default 2>/dev/null || true

# 配置默认存储池
echo ""
echo "=== Configuring default storage pool ==="
if ! virsh pool-info default &>/dev/null; then
    echo "Defining default storage pool..."
    virsh pool-define-as default dir --target /var/lib/libvirt/images || true
fi

if virsh pool-info default 2>/dev/null | grep -q "State:.*inactive"; then
    echo "Starting default storage pool..."
    virsh pool-start default || true
fi

virsh pool-autostart default 2>/dev/null || true

# 显示状态（允许失败，仅用于日志）
echo ""
echo "=== Network status ==="
virsh net-list --all || echo "[WARN] Failed to list networks"

echo ""
echo "=== Storage pools ==="
virsh pool-list --all || echo "[WARN] Failed to list storage pools"

# 启动 supervisord（管理 JVP 进程）
echo ""
echo "=== Starting supervisord ==="
exec /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf
