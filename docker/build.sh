#!/bin/bash
# build.sh - 构建 JVP Docker 镜像（包含 libvirtd）
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Building JVP Docker Image ==="
echo "Project directory: $PROJECT_DIR"

cd "$PROJECT_DIR"

# 1. 构建 Go 二进制文件
echo ""
echo "=== Step 1: Building JVP binary ==="
if command -v task &> /dev/null; then
    task build
elif [ -f "Taskfile.yml" ]; then
    echo "Please install 'task' or run: go build -o jvp ./cmd/jvp"
    go build -o jvp ./cmd/jvp
else
    go build -o jvp ./cmd/jvp
fi

# 检查二进制文件
if [ ! -f "jvp" ]; then
    echo "[ERROR] JVP binary not found"
    exit 1
fi
echo "[OK] JVP binary built successfully"

# 2. 构建 Docker 镜像
echo ""
echo "=== Step 2: Building Docker image ==="
docker build -f Dockerfile.libvirt -t jvp:libvirt .

echo ""
echo "=== Build completed ==="
echo ""
echo "To run the container:"
echo "  docker-compose up -d"
echo ""
echo "Or manually:"
echo "  docker run -d --privileged \\"
echo "    --device /dev/kvm \\"
echo "    --device /dev/net/tun \\"
echo "    -p 7777:7777 \\"
echo "    -v jvp-libvirt:/var/lib/libvirt \\"
echo "    -v jvp-data:/app/data \\"
echo "    --name jvp \\"
echo "    jvp:libvirt"
echo ""
echo "Access JVP at: http://localhost:7777"
