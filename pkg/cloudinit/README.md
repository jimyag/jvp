# Cloud-Init 包

一个独立的、可复用的 cloud-init 配置生成器包。

## 特性

- ✅ **类型安全**: 使用 Go 结构体定义配置，编译时检查
- ✅ **标准化**: 直接序列化为标准 cloud-init YAML 格式
- ✅ **模块化**: 独立的包，可在任何项目中使用
- ✅ **完整功能**: 支持所有 cloud-init 配置选项
- ✅ **易于测试**: 纯函数式设计，易于单元测试
- ✅ **ISO 生成**: 内置 ISO 构建器

## 安装

```bash
go get github.com/jimyag/jvp/pkg/cloudinit
```

## 快速开始

### 方式 1：直接使用 UserData 结构（推荐，最灵活）

```go
package main

import (
    "fmt"
    "log"

    "github.com/jimyag/jvp/pkg/cloudinit"
)

func main() {
    gen := cloudinit.NewGenerator()

    lockPasswd := false
    // 直接构建标准的 UserData 结构
    userData := &cloudinit.UserData{
        Groups: map[string][]string{
            "developers": {"john", "jane"},
        },
        Users: []any{
            "default",
            cloudinit.User{
                Name:              "admin",
                Groups:            "sudo",
                LockPasswd:        &lockPasswd,
                SSHAuthorizedKeys: []string{"ssh-ed25519 AAAA..."},
            },
        },
        Packages: []string{"docker.io", "git"},
        RunCmd:   []string{"systemctl enable docker"},
    }

    // 直接序列化为 YAML
    content, err := gen.GenerateUserDataFromStruct(userData)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(content)
}
```

### 方式 2：使用 Config 结构（高级封装）

```go
package main

import (
    "fmt"
    "log"

    "github.com/jimyag/jvp/pkg/cloudinit"
)

func main() {
    gen := cloudinit.NewGenerator()

    lockPasswd := false
    // 使用 Config 结构，自动处理密码哈希等
    config := &cloudinit.Config{
        Hostname: "my-server",
        Users: []cloudinit.User{
            {
                Name:              "admin",
                Groups:            "sudo",
                PlainTextPasswd:   "password123", // 自动哈希
                LockPasswd:        &lockPasswd,
                SSHAuthorizedKeys: []string{"ssh-ed25519 AAAA..."},
            },
        },
        Packages: []string{"docker.io", "git"},
        Commands: []string{"systemctl enable docker"},
    }

    userData, err := gen.GenerateUserData(config)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(userData)
}
```

输出：

```yaml
#cloud-config
users:
  - default
  - name: admin
    groups: sudo
    lock_passwd: false
    passwd: $2a$10$...
    ssh_authorized_keys:
      - ssh-ed25519 AAAA...
packages:
  - docker.io
  - git
runcmd:
  - systemctl enable docker
```

### 生成完整的 cloud-init ISO

```go
package main

import (
    "log"

    "github.com/jimyag/jvp/pkg/cloudinit"
)

func main() {
    builder := cloudinit.NewISOBuilder()

    lockPasswd := false
    isoPath, err := builder.BuildISO(&cloudinit.BuildOptions{
        VMName:    "my-vm",
        OutputDir: "/var/lib/jvp/images",
        Config: &cloudinit.Config{
            Hostname: "my-server",
            Users: []cloudinit.User{
                {
                    Name:            "admin",
                    Groups:          "sudo",
                    PlainTextPasswd: "admin123",
                    LockPasswd:      &lockPasswd,
                },
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("ISO created: %s", isoPath)
}
```

## 核心组件

### Generator

配置生成器，负责生成 cloud-init 配置文件的内容。

```go
gen := cloudinit.NewGenerator()

// 方式 1：直接从结构序列化（最灵活）
userData, _ := gen.GenerateUserDataFromStruct(&cloudinit.UserData{
    Users: []any{"default"},
    Packages: []string{"vim"},
})

networkConfig, _ := gen.GenerateNetworkConfigFromStruct(&cloudinit.NetworkData{
    Version: "2",
    Ethernets: map[string]cloudinit.Ethernet{
        "eth0": {DHCP4: true},
    },
})

// 方式 2：从 Config 生成（自动处理密码等）
userData, _ = gen.GenerateUserData(config)
networkConfig, _ = gen.GenerateNetworkConfig(network)

// 生成 meta-data
metaData, _ := gen.GenerateMetaData("my-server")
```

### ISOBuilder

ISO 构建器，负责生成可挂载的 cloud-init ISO 镜像。

```go
builder := cloudinit.NewISOBuilder()

// 构建 ISO
isoPath, _ := builder.BuildISO(&cloudinit.BuildOptions{
    VMName: "my-vm",
    Config: config,
})

// 清理 ISO
_ = builder.CleanupISO("my-vm", "")

// 获取 ISO 路径
path := builder.GetISOPath("my-vm", "")
```

## 配置结构

### Config

主配置结构，包含所有 cloud-init 选项：

```go
type Config struct {
    Hostname       string   // 主机名
    Users          []User   // 用户列表
    Groups         []Group  // 组列表
    DisableRoot    bool     // 禁用 root
    Network        *Network // 网络配置
    Commands       []string // 启动命令
    Packages       []string // 软件包
    WriteFiles     []File   // 写入文件
    Timezone       string   // 时区
    CustomUserData string   // 自定义 YAML
}
```

### User

用户配置，支持所有 cloud-init 用户选项：

```go
type User struct {
    Name              string      // 用户名
    Gecos             string      // 全名
    Groups            string      // 组（逗号分隔）
    Shell             string      // Shell
    Sudo              interface{} // sudo 规则
    LockPasswd        *bool       // 锁定密码
    PlainTextPasswd   string      // 明文密码（自动 hash）
    SSHAuthorizedKeys []string    // SSH 公钥
    SSHImportID       []string    // 导入 SSH 密钥
    // ... 更多字段
}
```

### Network

网络配置，支持 cloud-init network config v2：

```go
type Network struct {
    Version   string              // 版本（默认："2"）
    Ethernets map[string]Ethernet // 网卡配置
}

type Ethernet struct {
    DHCP4     bool     // DHCP4
    Addresses []string // 静态 IP
    Gateway4  string   // 网关
    Nameservers struct {
        Addresses []string // DNS
    }
}
```

## 高级用法

### 使用 UserData 直接序列化（完全控制）

直接使用 `UserData` 结构可以完全控制生成的 YAML：

```go
gen := cloudinit.NewGenerator()

lockPasswd := false
sshPwauth := true

userData := &cloudinit.UserData{
    Groups: map[string][]string{
        "developers": {"john", "jane"},
    },
    Users: []any{
        "default",
        cloudinit.User{
            Name:   "admin",
            Groups: "sudo",
            LockPasswd: &lockPasswd,
        },
    },
    SSHPwauth: &sshPwauth,
    Packages: []string{"vim", "git"},
    RunCmd: []string{"apt-get update"},
    FinalMessage: "Setup completed!",
    PowerState: &cloudinit.PowerState{
        Mode:    "reboot",
        Message: "Rebooting...",
        Timeout: 30,
    },
}

content, _ := gen.GenerateUserDataFromStruct(userData)
```

### 网桥和高级网络配置

使用 `NetworkData` 直接配置复杂的网络拓扑：

```go
gen := cloudinit.NewGenerator()

br0 := cloudinit.Bridge{
    Interfaces: []string{"eth0", "eth1"},
    DHCP4:      false,
    Addresses:  []string{"10.0.0.1/24"},
    Gateway4:   "10.0.0.254",
}
br0.Nameservers.Addresses = []string{"8.8.8.8"}

networkData := &cloudinit.NetworkData{
    Version: "2",
    Bridges: map[string]cloudinit.Bridge{
        "br0": br0,
    },
}

content, _ := gen.GenerateNetworkConfigFromStruct(networkData)
```

### 导入 SSH 密钥

从 GitHub 或 Launchpad 导入 SSH 密钥：

```go
User{
    Name: "developer",
    SSHImportID: []string{
        "gh:username",  // GitHub
        "lp:username",  // Launchpad
    },
}
```

### 复杂的 sudo 规则

```go
// 字符串形式
User{
    Name: "admin",
    Sudo: "ALL=(ALL) NOPASSWD:ALL",
}

// 数组形式
User{
    Name: "developer",
    Sudo: []string{
        "ALL=(ALL) NOPASSWD:/usr/bin/docker",
        "ALL=(ALL) NOPASSWD:/usr/bin/systemctl",
    },
}

// 禁用 sudo
User{
    Name: "guest",
    Sudo: false,
}
```

### 系统用户

创建无家目录的系统用户：

```go
User{
    Name:   "appuser",
    System: true,
    Shell:  "/usr/sbin/nologin",
}
```

### 账户过期

设置账户过期日期：

```go
User{
    Name:       "tempuser",
    Expiredate: "2025-12-31",
    Inactive:   "5", // 密码过期后 5 天禁用
}
```

### 自定义 user-data

如果需要完全自定义 user-data：

```go
config := &cloudinit.Config{
    CustomUserData: `#cloud-config
users:
  - name: custom
    gecos: Custom User
packages:
  - nginx
`,
}
```

## 密码哈希

使用 `HashPassword` 函数生成 bcrypt 哈希：

```go
hash, err := cloudinit.HashPassword("password123")
// hash: $2a$10$...
```

或使用命令行工具：

```bash
mkpasswd --method=SHA-512 --rounds=4096
```

## 向后兼容

支持简化的配置方式（已废弃，建议使用新配置）：

```go
config := &cloudinit.Config{
    Hostname: "my-server",
    Username: "admin",     // 已废弃
    Password: "password",  // 已废弃
    SSHKeys:  []string{}, // 已废弃
}
```

## 架构设计

### 数据流

```
Config (用户配置)
    ↓
Generator.GenerateUserData()
    ↓
UserData (内部结构)
    ↓
yaml.Marshal()
    ↓
YAML 字符串
    ↓
写入文件 → ISO
```

### 关键特性

1. **类型安全**: 所有配置都是强类型的 Go 结构体
2. **序列化**: 使用 `yaml.Marshal()` 自动序列化
3. **标签支持**: 使用 `yaml` 标签控制输出格式
4. **omitempty**: 自动省略空值字段
5. **接口灵活性**: `Sudo` 字段支持多种类型

## 测试

运行测试：

```bash
go test ./pkg/cloudinit/... -v
```

示例测试：

```go
func TestGenerateUserData(t *testing.T) {
    gen := cloudinit.NewGenerator()

    userData, err := gen.GenerateUserData(&cloudinit.Config{
        Hostname: "test-server",
        Users: []cloudinit.User{
            {Name: "test", Groups: "sudo"},
        },
    })

    if err != nil {
        t.Fatal(err)
    }

    if !strings.Contains(userData, "#cloud-config") {
        t.Error("Missing cloud-config header")
    }
}
```

## 依赖

- `gopkg.in/yaml.v3` - YAML 序列化
- `golang.org/x/crypto/bcrypt` - 密码哈希

## 许可证

与主项目相同

## 贡献

欢迎贡献！请确保：

1. 添加测试覆盖新功能
2. 更新文档
3. 遵循代码风格
4. 通过所有测试

## 相关链接

- [Cloud-Init 官方文档](https://cloudinit.readthedocs.io/)
- [Cloud-Init 用户配置示例](https://cloudinit.readthedocs.io/en/latest/reference/examples.html)
- [Cloud-Init 网络配置 v2](https://cloudinit.readthedocs.io/en/latest/reference/network-config-format-v2.html)
