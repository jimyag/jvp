# 故障排查指南

## 常见错误及解决方法

### 1. "Unexpected end of JSON input"

**错误描述**: 前端显示 `SyntaxError: Unexpected end of JSON input`

**可能原因**:
- 后端 API 返回空响应
- 后端没有运行
- 后端返回 HTML 错误页面而不是 JSON
- API 路径错误

**解决步骤**:

1. **检查后端是否运行**
   ```bash
   curl http://192.168.2.100:8080/api/volumes/describe \
     -X POST \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

   如果无法连接,启动后端:
   ```bash
   # 在 jvp 项目根目录
   go run cmd/main.go
   ```

2. **检查 API 路径是否正确**

   参考 `web/API_MAPPING.md` 确认 API 路径。

   正确的路径:
   - ✅ `/api/volumes/describe`
   - ❌ `/api/volumes`

3. **检查浏览器控制台**

   打开开发者工具 (F12),查看 Console 和 Network 标签页:
   - 查看请求状态码 (应该是 200)
   - 查看响应内容 (应该是 JSON)
   - 查看是否有 CORS 错误

4. **检查后端配置**

   确认 `web/next.config.ts` 中的后端地址正确:
   ```typescript
   destination: 'http://192.168.2.100:8080/api/:path*'
   ```

### 2. 连接被拒绝 (Connection Refused)

**错误描述**: `ERR_CONNECTION_REFUSED` 或 `Failed to fetch`

**解决步骤**:

1. **检查后端端口**
   ```bash
   # 检查 8080 端口是否监听
   lsof -i :8080
   # 或
   netstat -an | grep 8080
   ```

2. **检查防火墙**
   ```bash
   # macOS
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

   # Linux
   sudo ufw status
   sudo firewall-cmd --list-all
   ```

3. **测试网络连通性**
   ```bash
   ping 192.168.2.100
   telnet 192.168.2.100 8080
   ```

### 3. CORS 错误

**错误描述**: `Access to fetch has been blocked by CORS policy`

**解决方法**:

后端需要配置 CORS 中间件。在 Gin 中添加:

```go
import "github.com/gin-contrib/cors"

// 在创建 engine 后添加
engine.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"POST", "OPTIONS"},
    AllowHeaders:     []string{"Content-Type"},
    AllowCredentials: true,
}))
```

### 4. 404 Not Found

**错误描述**: API 返回 404 状态码

**可能原因**:
- API 路径拼写错误
- 后端路由未注册
- HTTP 方法错误

**检查清单**:
- [ ] 确认使用 POST 方法 (不是 GET/DELETE)
- [ ] 确认路径包含 `/api/` 前缀
- [ ] 确认子路径正确 (`/describe`, `/create` 等)
- [ ] 查看后端启动日志中的路由列表

### 5. 500 Internal Server Error

**错误描述**: API 返回 500 状态码

**解决步骤**:

1. **查看后端日志**
   ```bash
   # 后端会打印详细的错误信息
   # 查找包含 "ERROR" 或 "Failed" 的日志行
   ```

2. **常见后端错误**:
   - 数据库连接失败
   - libvirt 连接失败
   - 请求参数格式错误
   - 资源不存在 (如镜像 ID 不存在)

3. **检查请求体格式**

   确保请求体字段名正确:
   ```json
   // ✅ 正确
   { "instance_ids": ["id1"] }

   // ❌ 错误
   { "instanceIds": ["id1"] }
   ```

### 6. 数据无法显示

**症状**: 页面加载成功,但表格中没有数据

**检查步骤**:

1. **查看网络请求**
   - 打开开发者工具 Network 标签
   - 刷新页面
   - 检查 API 请求是否成功 (200 状态)
   - 查看响应数据结构

2. **检查响应格式**

   后端应该返回:
   ```json
   {
     "volumes": [...]  // 注意字段名
   }
   ```

   而不是:
   ```json
   {
     "data": { "volumes": [...] }
   }
   ```

3. **检查字段映射**

   前端代码:
   ```typescript
   setVolumes(data.volumes || []);
   ```

   确保字段名匹配后端响应。

### 7. 前端启动失败

**错误**: `npm run dev` 失败

**解决步骤**:

1. **清除缓存重新安装**
   ```bash
   cd web
   rm -rf node_modules .next
   npm install
   ```

2. **检查 Node.js 版本**
   ```bash
   node --version  # 应该 >= 18
   ```

3. **检查端口占用**
   ```bash
   lsof -i :3000
   # 如果被占用,杀掉进程或换端口
   PORT=3001 npm run dev
   ```

## 调试技巧

### 使用浏览器开发者工具

1. **Network 标签页**
   - 查看所有 API 请求
   - 检查请求头、请求体
   - 查看响应状态码、响应体
   - 查看请求耗时

2. **Console 标签页**
   - 查看 console.log 输出
   - 查看错误堆栈
   - 查看警告信息

3. **Application 标签页**
   - 清除缓存
   - 查看 LocalStorage/SessionStorage

### 使用 curl 测试后端

```bash
# 测试查询接口
curl -v http://192.168.2.100:8080/api/volumes/describe \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{}'

# -v 参数显示详细信息
# 检查:
# - HTTP 状态码
# - 响应头
# - 响应体
```

### 查看后端日志

后端使用 zerolog,日志格式为 JSON:

```bash
# 运行后端
go run cmd/main.go 2>&1 | jq .

# jq 可以格式化 JSON 日志
# 查找特定请求:
go run cmd/main.go 2>&1 | grep "DescribeVolumes"
```

### 启用前端详细日志

在浏览器控制台运行:

```javascript
localStorage.setItem('debug', '*');
location.reload();
```

## 性能问题

### 页面加载慢

1. **检查网络延迟**
   ```bash
   ping 192.168.2.100
   # 延迟应该 < 10ms (局域网)
   ```

2. **检查后端响应时间**
   - 在 Network 标签查看 API 请求耗时
   - 如果 > 1秒,检查后端性能
   - 查看后端是否有慢查询

3. **优化建议**:
   - 使用分页加载大量数据
   - 实现缓存机制
   - 减少不必要的 API 调用

### 实时更新延迟

当前实现使用轮询刷新。如需实时更新:
- 实现 WebSocket
- 使用 Server-Sent Events (SSE)
- 减小轮询间隔

## 获取帮助

如果上述方法无法解决问题:

1. **收集信息**:
   - 错误截图
   - 浏览器控制台完整日志
   - 后端日志
   - 使用的浏览器和版本
   - 前后端版本

2. **提交 Issue**:
   - 到 GitHub 仓库提交 issue
   - 包含上述收集的信息
   - 描述复现步骤

3. **查看文档**:
   - `README.md` - 项目概览
   - `QUICKSTART.md` - 快速开始
   - `USAGE.md` - 使用指南
   - `API_MAPPING.md` - API 映射
