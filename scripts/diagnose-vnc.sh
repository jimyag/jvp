#!/bin/bash
# VNC 控制台诊断脚本

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <instance_id>"
    echo "Example: $0 i-98811441645093476"
    exit 1
fi

INSTANCE_ID=$1
VNC_SOCKET="/var/lib/jvp/qemu/${INSTANCE_ID}.vnc"

echo "=== VNC Console Diagnostics for $INSTANCE_ID ==="
echo

# 1. 检查 VNC socket 文件
echo "1. Checking VNC socket file:"
if [ -S "$VNC_SOCKET" ]; then
    echo "   ✓ Socket exists: $VNC_SOCKET"
    ls -la "$VNC_SOCKET"
else
    echo "   ✗ Socket does NOT exist: $VNC_SOCKET"
    echo "   Searching for VNC sockets..."
    find /var/lib/jvp/qemu -name "*.vnc" 2>/dev/null || true
    find /var/run/libvirt/qemu -name "*.vnc" 2>/dev/null || true
fi
echo

# 2. 检查虚拟机状态
echo "2. Checking VM status:"
virsh list --all | grep "$INSTANCE_ID" || echo "   VM not found in virsh"
echo

# 3. 检查虚拟机 XML 配置
echo "3. Checking VM XML for graphics config:"
if virsh dumpxml "$INSTANCE_ID" 2>/dev/null | grep -A 5 "<graphics"; then
    echo "   ✓ Graphics config found"
else
    echo "   ✗ No graphics config found"
fi
echo

# 4. 测试 socket 连接
echo "4. Testing socket connection:"
if [ -S "$VNC_SOCKET" ]; then
    if timeout 2 nc -U "$VNC_SOCKET" </dev/null 2>&1; then
        echo "   ✓ Socket is connectable"
    else
        echo "   ? Socket exists but connection test inconclusive"
    fi
else
    echo "   ✗ Cannot test - socket doesn't exist"
fi
echo

# 5. 检查进程
echo "5. Checking QEMU process:"
ps aux | grep "[q]emu.*$INSTANCE_ID" || echo "   No QEMU process found"
echo

# 6. 检查权限
echo "6. Checking current user permissions:"
echo "   Current user: $(whoami)"
echo "   Groups: $(groups)"
if [ -S "$VNC_SOCKET" ]; then
    echo "   Can read socket: $(test -r "$VNC_SOCKET" && echo YES || echo NO)"
    echo "   Can write socket: $(test -w "$VNC_SOCKET" && echo YES || echo NO)"
fi
echo

# 7. 检查 libvirt 连接
echo "7. Checking libvirt connection:"
virsh version 2>/dev/null || echo "   Cannot connect to libvirt"
echo

echo "=== Diagnostic complete ==="
