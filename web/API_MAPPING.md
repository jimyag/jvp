# API 映射文档

本文档说明前端与后端 API 的对应关系。

## 后端 API 设计

JVP 后端使用 **POST + 子路径** 的设计模式,所有 API 都是 POST 请求。

### 实例管理 (Instances)

| 功能 | HTTP方法 | 后端路径 | 请求体 |
|------|---------|---------|--------|
| 查询实例 | POST | `/api/instances/describe` | `{}` 或 `{ instance_ids: [...] }` |
| 创建实例 | POST | `/api/instances/run` | `{ name, vcpus, memory, disk, image_id, keypair_name }` |
| 启动实例 | POST | `/api/instances/start` | `{ instance_ids: ["id1", "id2"] }` |
| 停止实例 | POST | `/api/instances/stop` | `{ instance_ids: ["id1"] }` |
| 重启实例 | POST | `/api/instances/reboot` | `{ instance_ids: ["id1"] }` |
| 终止实例 | POST | `/api/instances/terminate` | `{ instance_ids: ["id1"] }` |
| 修改属性 | POST | `/api/instances/modify-attribute` | `{ instance_id, vcpus, memory }` |
| 重置密码 | POST | `/api/instances/reset-password` | `{ instance_id, password }` |

### 卷管理 (Volumes)

| 功能 | HTTP方法 | 后端路径 | 请求体 |
|------|---------|---------|--------|
| 查询卷 | POST | `/api/volumes/describe` | `{}` 或 `{ volume_ids: [...] }` |
| 创建卷 | POST | `/api/volumes/create` | `{ name, size, snapshot_id? }` |
| 删除卷 | POST | `/api/volumes/delete` | `{ volume_id }` |
| 附加卷 | POST | `/api/volumes/attach` | `{ volume_id, instance_id }` |
| 分离卷 | POST | `/api/volumes/detach` | `{ volume_id, instance_id? }` |
| 修改卷 | POST | `/api/volumes/modify` | `{ volume_id, size }` |

### 镜像管理 (Images)

| 功能 | HTTP方法 | 后端路径 | 请求体 |
|------|---------|---------|--------|
| 查询镜像 | POST | `/api/images/describe` | `{}` 或 `{ image_ids: [...] }` |
| 创建镜像 | POST | `/api/images/create` | `{ instance_id, image_name, description? }` |
| 注册镜像 | POST | `/api/images/register` | `{ name, url, os_type, description? }` |
| 注销镜像 | POST | `/api/images/deregister` | `{ image_id }` |

### 密钥对管理 (KeyPairs)

| 功能 | HTTP方法 | 后端路径 | 请求体 |
|------|---------|---------|--------|
| 查询密钥对 | POST | `/api/keypairs/describe` | `{}` 或 `{ keypair_names: [...] }` |
| 创建密钥对 | POST | `/api/keypairs/create` | `{ name, key_type: "rsa"\|"ed25519" }` |
| 导入密钥对 | POST | `/api/keypairs/import` | `{ name, public_key }` |
| 删除密钥对 | POST | `/api/keypairs/delete` | `{ keypair_id }` |

## 前端实现

前端已全部适配后端的 API 设计:

### 查询数据
```typescript
// 所有列表查询都使用 POST /api/{resource}/describe
const response = await fetch("/api/instances/describe", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({}), // 空对象表示查询所有
});
```

### 创建资源
```typescript
// 创建实例
await fetch("/api/instances/run", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    name: "my-instance",
    vcpus: 2,
    memory: 2048,
    disk: 20,
    image_id: "ubuntu-22.04",
  }),
});
```

### 操作资源
```typescript
// 启动实例
await fetch("/api/instances/start", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    instance_ids: ["instance-123"],
  }),
});
```

### 删除资源
```typescript
// 删除卷
await fetch("/api/volumes/delete", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    volume_id: "vol-123",
  }),
});
```

## 响应格式

所有 API 响应都遵循统一格式:

### 成功响应
```json
{
  "instances": [...],     // 列表查询
  "instance": {...},      // 单个资源
  "volumes": [...],
  "images": [...],
  "keypairs": [...],
  "private_key": "...",   // 创建密钥对时返回
  "return": true          // 删除操作返回
}
```

### 错误响应
```json
{
  "error": "错误信息"
}
```

## 与 RESTful 的区别

传统 RESTful 设计 vs JVP 后端设计:

| 操作 | RESTful | JVP 后端 |
|------|---------|----------|
| 列表查询 | GET /api/instances | POST /api/instances/describe |
| 创建资源 | POST /api/instances | POST /api/instances/run |
| 启动实例 | POST /api/instances/:id/start | POST /api/instances/start |
| 删除资源 | DELETE /api/instances/:id | POST /api/instances/terminate |

## 注意事项

1. **所有请求都是 POST**: 不使用 GET、PUT、DELETE 等 HTTP 方法
2. **资源 ID 在请求体中**: 不在 URL 路径中使用资源 ID
3. **批量操作支持**: 大多数操作支持传入多个 ID 数组
4. **Content-Type 必须**: 所有 POST 请求必须设置 `Content-Type: application/json`

## 测试示例

使用 curl 测试 API:

```bash
# 查询所有实例
curl -X POST http://192.168.2.100:8080/api/instances/describe \
  -H "Content-Type: application/json" \
  -d '{}'

# 创建实例
curl -X POST http://192.168.2.100:8080/api/instances/run \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-vm",
    "vcpus": 2,
    "memory": 2048,
    "disk": 20,
    "image_id": "ubuntu-22.04"
  }'

# 启动实例
curl -X POST http://192.168.2.100:8080/api/instances/start \
  -H "Content-Type: application/json" \
  -d '{
    "instance_ids": ["instance-abc123"]
  }'

# 查询所有卷
curl -X POST http://192.168.2.100:8080/api/volumes/describe \
  -H "Content-Type: application/json" \
  -d '{}'

# 创建密钥对
curl -X POST http://192.168.2.100:8080/api/keypairs/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-key",
    "key_type": "rsa"
  }'
```

## 前端配置

确保 `web/next.config.ts` 配置正确的后端地址:

```typescript
const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://192.168.2.100:8080/api/:path*',
      },
    ];
  },
};
```

所有前端 `/api/*` 请求会被代理到后端 `http://192.168.2.100:8080/api/*`。
