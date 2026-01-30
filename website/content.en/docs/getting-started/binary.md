---
title: Binary Deployment
weight: 3
---

# Binary Deployment

Deploy JVP using pre-built binaries from GitHub Releases.

## Step 1: Download the Binary

Download the binary for your system from [GitHub Releases](https://github.com/jimyag/jvp/releases).

```bash
# Create directory
sudo mkdir -p /opt/jvp

# Download and extract (example for linux amd64)
wget https://github.com/jimyag/jvp/releases/latest/download/jvp_linux_amd64.tar.gz
tar -xzf jvp_linux_amd64.tar.gz -C /opt/jvp
```

## Step 2: Create Systemd Service

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

## Step 3: Start the Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable jvp
sudo systemctl start jvp
```

## Step 4: Access Web Interface

Open your browser and navigate to:

```
http://<server-ip>:7777
```
