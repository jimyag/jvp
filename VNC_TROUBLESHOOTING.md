# VNC 控制台黑屏问题排查

## 问题现象

- 前端显示 "VNC console connected"
- 但是画面一直是黑屏
- 没有显示虚拟机的图形界面

## 可能的原因

### 1. VNC Socket 路径问题

**检查方法：**
```bash
# 在服务器上运行诊断脚本
./scripts/diagnose-vnc.sh i-98811441645093476

# 或手动检查
ls -la /var/lib/jvp/qemu/*.vnc
```

**常见问题：**
- Socket 文件不存在
- Socket 路径不正确
- 权限不足

### 2. VNC 数据未转发

**检查后端日志：**
```bash
# 查看 jvp 服务日志，应该看到：
# - "Starting VNC proxy" - 代理启动
# - "VNC->WS forwarding data" - 数据从 VNC 转发到 WebSocket
# - "WS->VNC forwarding data" - 数据从 WebSocket 转发到 VNC
```

**如果没有看到转发日志：**
- VNC socket 可能没有数据输出
- 虚拟机的 VNC 服务可能没有启动

### 3. VNC 握手失败

VNC 协议需要正确的握手过程：
1. 服务器发送版本字符串 (例如 "RFB 003.008\n")
2. 客户端响应版本
3. 安全协商
4. 初始化

**测试 VNC Socket：**
```bash
# 编译测试工具
cd scripts
go build -o test-vnc-socket test-vnc-socket.go

# 运行测试
sudo ./test-vnc-socket /var/lib/jvp/qemu/i-98811441645093476.vnc
```

**期望输出：**
```
✓ Socket file exists
✓ Successfully connected to socket
✓ Received 12 bytes from VNC server
  Data: "RFB 003.008\n"
✓ Valid VNC handshake detected
```

### 4. 虚拟机图形配置问题

**检查虚拟机 XML：**
```bash
virsh dumpxml i-98811441645093476 | grep -A 10 graphics
```

**正确的配置应该是：**
```xml
<graphics type='vnc' socket='/var/lib/jvp/qemu/i-98811441645093476.vnc'>
  <listen type='socket' socket='/var/lib/jvp/qemu/i-98811441645093476.vnc'/>
</graphics>
```

**错误的配置：**
```xml
<!-- 不要使用 port，应该使用 socket -->
<graphics type='vnc' port='5900' listen='0.0.0.0'/>
```

### 5. 虚拟机显示问题

虚拟机本身可能没有图形输出：

**可能原因：**
- 虚拟机没有安装图形界面（只有命令行）
- 虚拟机卡在启动画面
- 虚拟机的显卡配置有问题

**检查虚拟机视频设备：**
```bash
virsh dumpxml i-98811441645093476 | grep -A 5 video
```

**推荐配置：**
```xml
<video>
  <model type='qxl' ram='65536' vram='65536' vgamem='16384' heads='1' primary='yes'/>
  <address type='pci' domain='0x0000' bus='0x00' slot='0x02' function='0x0'/>
</video>
```

### 6. WebSocket 数据格式问题

noVNC 只接受二进制消息，检查代码：

**pkg/wsproxy/vnc_proxy.go:**
```go
// 确保发送 BinaryMessage
err = p.wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
```

**pkg/wsproxy/vnc_proxy.go:**
```go
// 确保只处理 BinaryMessage
if messageType == websocket.BinaryMessage {
    _, err = p.unixConn.Write(data)
}
```

## 诊断步骤

### 第一步：验证 Socket 存在

```bash
sudo ls -la /var/lib/jvp/qemu/i-98811441645093476.vnc
```

如果不存在，检查：
1. 实例 ID 是否正确
2. 虚拟机是否真的在运行
3. VNC 是否配置为使用 socket

### 第二步：测试 Socket 连接

```bash
sudo ./scripts/test-vnc-socket.go /var/lib/jvp/qemu/i-98811441645093476.vnc
```

如果连接失败：
- 检查权限（jvp 进程的运行用户）
- 检查 libvirt 配置

### 第三步：检查后端日志

启用 Debug 级别日志并访问 VNC 控制台，查看：

```bash
# 应该看到：
[INFO] Starting VNC proxy vnc_socket=/var/lib/jvp/qemu/xxx.vnc
[DEBUG] VNC->WS forwarding data bytes=12 total=12
[DEBUG] WS->VNC forwarding data bytes=12 total=12
...
```

如果只有 "Starting VNC proxy" 没有数据转发：
- VNC socket 可能没有发送握手数据
- 检查虚拟机是否真的在运行

### 第四步：检查浏览器控制台

打开浏览器开发者工具 (F12)，查看 Console 标签：

**正常情况：**
```
VNC connected
```

**异常情况：**
```
VNC security failure: ...
VNC disconnected: ...
WebSocket connection failed: ...
```

### 第五步：检查虚拟机状态

```bash
# 查看虚拟机是否真的在运行
virsh list | grep i-98811441645093476

# 如果是关机状态，启动它
virsh start i-98811441645093476

# 查看虚拟机控制台输出（文本模式）
virsh console i-98811441645093476
# (按 Ctrl+] 退出)
```

## 常见解决方案

### 解决方案 1：修复 Socket 路径

如果 socket 在不同位置：

```bash
# 查找实际的 socket 位置
sudo find /var -name "*.vnc" -type s 2>/dev/null

# 修改 libvirt 配置或代码中的路径
```

### 解决方案 2：修复权限

```bash
# 查看 jvp 进程的用户
ps aux | grep jvp

# 确保该用户在 libvirt 组中
sudo usermod -a -G libvirt jvp-user

# 或临时更改 socket 权限（不推荐）
sudo chmod 666 /var/lib/jvp/qemu/*.vnc
```

### 解决方案 3：重新配置虚拟机 VNC

编辑虚拟机 XML：

```bash
virsh edit i-98811441645093476
```

确保有：
```xml
<graphics type='vnc' socket='/var/lib/jvp/qemu/i-98811441645093476.vnc'>
  <listen type='socket' socket='/var/lib/jvp/qemu/i-98811441645093476.vnc'/>
</graphics>
```

然后重启虚拟机：
```bash
virsh shutdown i-98811441645093476
virsh start i-98811441645093476
```

### 解决方案 4：虚拟机没有图形界面

如果虚拟机是服务器版本（无 GUI），VNC 只会显示启动日志或登录提示。

**验证：**
使用 Serial Console 登录，检查是否安装了桌面环境：

```bash
# Ubuntu/Debian
dpkg -l | grep -i desktop

# 如果没有，可以安装（在虚拟机内）
sudo apt install ubuntu-desktop
```

## 调试技巧

### 启用详细日志

编辑后端代码，临时将日志级别改为 Debug：

```go
// 在 vnc_proxy.go 中
log.Debug() // 改为 log.Info()
```

### 使用 tcpdump 抓包

```bash
# 抓取 Unix Socket 通信（需要特殊工具）
sudo strace -e trace=read,write -p $(pgrep -f "jvp")
```

### 对比工作的 VNC 客户端

使用标准的 VNC 客户端测试：

```bash
# 安装 virt-viewer
sudo apt install virt-viewer

# 连接虚拟机（会自动找到 VNC）
virt-viewer -c qemu:///system i-98811441645093476
```

如果 virt-viewer 能显示但 Web 不能，问题在 WebSocket 代理。

## 预防措施

### 1. 虚拟机创建时确保配置 VNC

在 `instance.go` 中创建虚拟机时：

```go
graphics := Graphics{
    Type:   "vnc",
    Socket: fmt.Sprintf("/var/lib/jvp/qemu/%s.vnc", instanceID),
}
```

### 2. 监控日志

定期检查是否有 VNC 相关错误。

### 3. 健康检查

创建定期检查脚本：

```bash
#!/bin/bash
for vm in $(virsh list --name); do
    ./scripts/test-vnc-socket.go "/var/lib/jvp/qemu/${vm}.vnc"
done
```

## 需要帮助？

如果以上方法都无法解决问题，请收集以下信息：

1. **诊断脚本输出：**
   ```bash
   ./scripts/diagnose-vnc.sh i-xxx > debug.log 2>&1
   ```

2. **后端日志：**
   ```bash
   journalctl -u jvp -n 100 > jvp.log
   ```

3. **虚拟机 XML：**
   ```bash
   virsh dumpxml i-xxx > vm.xml
   ```

4. **浏览器控制台截图**

5. **VNC socket 测试结果：**
   ```bash
   sudo ./scripts/test-vnc-socket /path/to/socket.vnc
   ```
