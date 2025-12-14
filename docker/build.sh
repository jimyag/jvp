#!/bin/bash
# build.sh - 本地构建 JVP Docker 镜像
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Building JVP Docker Image ==="
cd "$PROJECT_DIR"

# 1. 构建 Go 二进制文件
echo ""
echo "=== Step 1: Building JVP binary ==="
if command -v task &> /dev/null; then
    task build
else
    go build -o jvp ./cmd/jvp
fi

if [ ! -f "jvp" ]; then
    echo "[ERROR] JVP binary not found"
    exit 1
fi
echo "[OK] JVP binary built"

# 2. 构建 Docker 镜像
echo ""
echo "=== Step 2: Building Docker image ==="
docker build -t ghcr.io/jimyag/jvp:latest .

echo ""
echo "=== Build completed ==="
echo "Run: docker-compose up -d"
echo "Access: http://localhost:7777"
