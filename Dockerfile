# Dockerfile for JVP
# GoReleaser 会在构建阶段处理前端构建和 Go 编译
# 这个 Dockerfile 只需要将构建好的二进制文件打包到镜像中

FROM alpine:latest

# 安装必要的运行时依赖
# 注意：如果需要连接 libvirt，可能需要额外的配置
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 jvp && \
    adduser -D -u 1000 -G jvp jvp

WORKDIR /app

# 从构建上下文复制二进制文件
# GoReleaser 会自动将构建好的二进制文件复制到这里
COPY jvp /app/jvp

# 设置权限
RUN chmod +x /app/jvp && \
    chown -R jvp:jvp /app

# 切换到非 root 用户
USER jvp

# 暴露端口（默认 7777）
EXPOSE 7777

# 设置环境变量
ENV JVP_ADDRESS=0.0.0.0:7777
ENV LIBVIRT_URI=qemu:///system
ENV JVP_DATA_DIR=/app/data

# 健康检查（检查根路径，如果服务正常运行应该可以访问）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:7777/ || exit 1

# 启动应用
ENTRYPOINT ["/app/jvp"]

