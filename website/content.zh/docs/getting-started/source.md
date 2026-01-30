---
title: 源码编译
weight: 4
---

# 源码编译

从源代码构建并运行 JVP。

## 前置条件

- Go 1.21+
- Node.js 18+
- Task（任务运行器）

## 步骤 1：克隆仓库

```bash
git clone https://github.com/jimyag/jvp.git
cd jvp
```

## 步骤 2：构建项目

```bash
# 构建包含前端的完整二进制文件
task build
```

## 步骤 3：运行服务

```bash
# 运行 JVP 服务（默认端口 7777）
./bin/jvp
```

## 步骤 4：访问 Web 界面

构建完成后，前端已嵌入到二进制文件中。访问：

```
http://localhost:7777
```

## 使用 Docker 本地调试

```bash
# 构建本地调试镜像
task debug-image

# 修改 docker-compose.yml 中的镜像为 jvp:local，然后启动
docker compose up -d
```
