# 将 Vite 前端打包到后端

## 概述

Vite 构建的前端应用会被嵌入到 Go 后端中，通过 `//go:embed` 功能实现。构建产物会被复制到 `internal/jvp/api/static/` 目录，然后编译到二进制文件中。

## 构建流程

### 方法 1: 使用 Makefile

```bash
# 构建前端并打包到后端
make build

# 或者只构建前端
make build-web
```

### 方法 2: 使用 Taskfile

```bash
# 构建当前系统架构的二进制文件（包含前端）
task build

# 构建特定架构
task build-linux-amd64
task build-linux-arm64
task build-darwin-amd64
task build-darwin-arm64

# 构建所有架构
task build-all
```

### 方法 3: 手动构建

```bash
# 1. 构建 Vite 前端
cd web-vite
npm install
npm run build

# 2. 复制构建产物到后端
rm -rf ../internal/jvp/api/static
mkdir -p ../internal/jvp/api/static
cp -r dist/* ../internal/jvp/api/static/

# 3. 构建后端（会自动嵌入前端文件）
cd ..
go build -o jvp ./cmd/jvp/
```

## 构建产物结构

Vite 构建后的文件结构：
```
web-vite/dist/
├── index.html          # 入口 HTML
├── assets/             # JS/CSS 文件（带 hash）
│   ├── index-*.js
│   ├── index-*.css
│   └── ...
├── novnc/              # VNC 相关文件
├── vendor/             # 第三方库
└── favicon.ico
```

这些文件会被复制到 `internal/jvp/api/static/`，然后通过 `//go:embed static/*` 嵌入到二进制文件中。

## 后端路由配置

后端已经配置好支持 Vite 的 SPA 路由：

- `/assets/*` - 静态资源（JS/CSS 文件）
- `/static/*` - 其他静态文件（如 novnc、vendor 等）
- `/*` - 所有其他路由都会回退到 `index.html`（SPA 路由）

## 注意事项

1. **构建顺序**：必须先构建前端，再构建后端，因为后端在编译时会嵌入前端文件。

2. **清理构建产物**：
   ```bash
   make clean
   # 或
   rm -rf web-vite/dist internal/jvp/api/static
   ```

3. **开发模式**：开发时可以直接运行 `cd web-vite && npm run dev`，不需要每次都打包到后端。

4. **生产部署**：生产环境使用 `make build` 或 `task build` 构建包含前端的完整二进制文件。

## 验证

构建完成后，可以验证前端是否已正确嵌入：

```bash
# 构建
make build

# 运行服务器
./jvp serve

# 访问 http://localhost:8080 应该能看到前端界面
```

