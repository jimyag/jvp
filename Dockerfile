# Dockerfile for JVP
# 包含 libvirtd 的完整虚拟化环境

FROM ubuntu:24.04

# 设置环境变量
ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Asia/Shanghai \
    LIBVIRT_URI=qemu:///system \
    JVP_ADDRESS=0.0.0.0:7777 \
    JVP_DATA_DIR=/app/data

# 安装必要的包
RUN apt-get update && apt-get install -y --no-install-recommends \
    # 虚拟化核心
    libvirt-daemon-system \
    libvirt-clients \
    qemu-system-x86 \
    qemu-utils \
    ovmf \
    # cloud-init ISO 生成
    genisoimage \
    # 网络工具
    dnsmasq \
    iptables \
    iproute2 \
    bridge-utils \
    net-tools \
    # SSH 客户端（远程节点连接）
    openssh-client \
    # 下载工具
    wget \
    curl \
    ca-certificates \
    # 进程管理
    supervisor \
    # 其他工具
    procps \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# 配置 libvirt
RUN mkdir -p /var/run/libvirt \
    && mkdir -p /var/lib/libvirt/images \
    && mkdir -p /var/lib/libvirt/qemu \
    && mkdir -p /etc/libvirt/qemu/networks

# 配置 QEMU 网桥权限
RUN mkdir -p /etc/qemu \
    && echo 'allow all' > /etc/qemu/bridge.conf \
    && chmod 644 /etc/qemu/bridge.conf

# 配置 libvirtd 允许非 root 访问
RUN sed -i 's/#unix_sock_group = "libvirt"/unix_sock_group = "libvirt"/' /etc/libvirt/libvirtd.conf \
    && sed -i 's/#unix_sock_rw_perms = "0770"/unix_sock_rw_perms = "0770"/' /etc/libvirt/libvirtd.conf

# 创建默认网络配置
RUN cat > /etc/libvirt/qemu/networks/default.xml << 'EOF'
<network>
  <name>default</name>
  <uuid>00000000-0000-0000-0000-000000000001</uuid>
  <forward mode='nat'>
    <nat>
      <port start='1024' end='65535'/>
    </nat>
  </forward>
  <bridge name='virbr0' stp='on' delay='0'/>
  <ip address='192.168.122.1' netmask='255.255.255.0'>
    <dhcp>
      <range start='192.168.122.2' end='192.168.122.254'/>
    </dhcp>
  </ip>
</network>
EOF

# 创建应用目录
WORKDIR /app
RUN mkdir -p /app/data

# 从构建上下文复制二进制文件
# GoReleaser 会自动将构建好的二进制文件复制到 $TARGETPLATFORM/ 目录
# 本地构建时直接复制 jvp 二进制文件
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM:+${TARGETPLATFORM}/}jvp /app/jvp
RUN chmod +x /app/jvp

# 复制 Docker 相关配置文件
COPY docker/supervisord.conf /etc/supervisor/conf.d/jvp.conf
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 暴露端口
EXPOSE 7777

# 存储卷
VOLUME ["/var/lib/libvirt", "/app/data"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:7777/ || exit 1

ENTRYPOINT ["/entrypoint.sh"]
