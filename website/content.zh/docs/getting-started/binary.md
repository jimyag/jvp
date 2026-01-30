---
title: 二进制部署
weight: 3
---

# 二进制部署

使用 GitHub Releases 中的预编译二进制文件部署 JVP。

## 步骤 1：下载二进制文件

从 [GitHub Releases](https://github.com/jimyag/jvp/releases) 下载适合你系统的二进制文件。

```bash
# 创建目录
sudo mkdir -p /opt/jvp

# 下载并解压（以 linux amd64 为例）
wget https://github.com/jimyag/jvp/releases/latest/download/jvp_linux_amd64.tar.gz
tar -xzf jvp_linux_amd64.tar.gz -C /opt/jvp
```

## 步骤 2：创建 Systemd 服务

```bash
sudo tee /etc/systemd/system/jvp.service > /dev/null <<EOF
[Unit]
Description=JVP - jimyag's virtualization platform
After=network.target libvirtd.service
Wants=network.target

[Service]
User=root
Group=root
Restart=always
ExecStart=/opt/jvp/jvp
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
```

## 步骤 3：启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable jvp
sudo systemctl start jvp
```

## 步骤 4：访问 Web 界面

打开浏览器访问：

```
http://<服务器IP>:7777
```
