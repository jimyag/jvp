package cloudinit

// Config cloud-init 配置
type Config struct {
	Hostname       string   // 主机名
	Users          []User   // 用户列表（如果为空，会创建默认用户）
	Groups         []Group  // 组列表
	DisableRoot    bool     // 禁用 root 登录（默认：true）
	Network        *Network // 网络配置（可选）
	Commands       []string // 启动后执行的命令
	Packages       []string // 要安装的软件包
	WriteFiles     []File   // 要写入的文件
	Timezone       string   // 时区（如：Asia/Shanghai）
	CustomUserData string   // 自定义 user-data YAML 内容（会覆盖其他配置）

	// 已废弃：为了向后兼容保留，建议使用 Users 字段
	Username string   // 用户名（默认：ubuntu）- 已废弃，请使用 Users
	Password string   // 用户密码（明文，会被 hash）- 已废弃，请使用 Users
	SSHKeys  []string // SSH 公钥列表 - 已废弃，请使用 Users
}

// User cloud-init 用户配置
type User struct {
	Name              string      `yaml:"name,omitempty"`                // 用户登录名
	Gecos             string      `yaml:"gecos,omitempty"`               // 用户全名/描述
	Homedir           string      `yaml:"homedir,omitempty"`             // 家目录路径
	PrimaryGroup      string      `yaml:"primary_group,omitempty"`       // 主组
	Groups            string      `yaml:"groups,omitempty"`              // 附加组，逗号分隔（如："users,admin"）
	SELinuxUser       string      `yaml:"selinux_user,omitempty"`        // SELinux 用户
	LockPasswd        *bool       `yaml:"lock_passwd,omitempty"`         // 锁定密码登录
	Passwd            string      `yaml:"passwd,omitempty"`              // 密码哈希（使用 hashPassword 生成）
	PlainTextPasswd   string      `yaml:"plain_text_passwd,omitempty"`   // 明文密码（不推荐）
	HashedPasswd      string      `yaml:"hashed_passwd,omitempty"`       // 密码哈希（同 Passwd）
	Sudo              interface{} `yaml:"sudo,omitempty"`                // sudo 规则：string, []string 或 false
	SSHAuthorizedKeys []string    `yaml:"ssh_authorized_keys,omitempty"` // SSH 公钥列表
	SSHImportID       []string    `yaml:"ssh_import_id,omitempty"`       // SSH 导入 ID（如："lp:username", "gh:username"）
	SSHRedirectUser   bool        `yaml:"ssh_redirect_user,omitempty"`   // 重定向 SSH 登录到默认用户
	Inactive          string      `yaml:"inactive,omitempty"`            // 密码过期后账户禁用的天数
	Expiredate        string      `yaml:"expiredate,omitempty"`          // 账户过期日期（YYYY-MM-DD）
	NoCreateHome      bool        `yaml:"no_create_home,omitempty"`      // 不创建家目录
	NoUserGroup       bool        `yaml:"no_user_group,omitempty"`       // 不创建与用户同名的组
	NoLogInit         bool        `yaml:"no_log_init,omitempty"`         // 不初始化 lastlog 和 faillog
	Shell             string      `yaml:"shell,omitempty"`               // 登录 Shell
	System            bool        `yaml:"system,omitempty"`              // 系统用户（无家目录）
	SnapUser          string      `yaml:"snapuser,omitempty"`            // Snappy 用户邮箱
}

// Group cloud-init 组配置
type Group struct {
	Name    string   // 组名
	Members []string // 组成员列表
}

// Network cloud-init 网络配置
type Network struct {
	Version   string              // 版本（默认：2）
	Ethernets map[string]Ethernet // 网卡配置
}

// Ethernet 以太网接口配置
type Ethernet struct {
	DHCP4       bool     `yaml:"dhcp4,omitempty"`     // 启用 DHCP4
	DHCP6       bool     `yaml:"dhcp6,omitempty"`     // 启用 DHCP6
	Addresses   []string `yaml:"addresses,omitempty"` // 静态 IP 地址（CIDR 格式，如：192.168.1.100/24）
	Gateway4    string   `yaml:"gateway4,omitempty"`  // IPv4 网关
	Gateway6    string   `yaml:"gateway6,omitempty"`  // IPv6 网关
	Nameservers struct {
		Addresses []string `yaml:"addresses,omitempty"` // DNS 服务器地址列表
	} `yaml:"nameservers,omitempty"` // DNS 配置
}

// File 要写入的文件
type File struct {
	Path        string // 文件路径
	Content     string // 文件内容
	Owner       string // 文件所有者（默认：root:root）
	Permissions string // 文件权限（默认：0644）
}

// ============================================================================
// 标准的 cloud-init 数据结构（可直接序列化为 YAML）
// ============================================================================

// MetaData 标准的 cloud-init meta-data 结构
// 可直接序列化为 YAML 格式
type MetaData struct {
	InstanceID    string `yaml:"instance-id"`
	LocalHostname string `yaml:"local-hostname"`
}

// UserData 标准的 cloud-init user-data 结构
// 可直接序列化为 YAML 格式，提供最大的灵活性
//
// 示例：
//
//	userData := &cloudinit.UserData{
//	    Groups: map[string][]string{
//	        "developers": {"john", "jane"},
//	    },
//	    Users: []any{
//	        "default",
//	        cloudinit.User{Name: "admin", Groups: "sudo"},
//	    },
//	    Packages: []string{"docker.io", "git"},
//	}
type UserData struct {
	Groups      map[string][]string `yaml:"groups,omitempty"`       // 组配置：map[组名] 组成员列表
	Users       []any               `yaml:"users,omitempty"`        // 用户列表：可包含 "default" 字符串和 User 对象
	DisableRoot bool                `yaml:"disable_root,omitempty"` // 禁用 root 登录
	Timezone    string              `yaml:"timezone,omitempty"`     // 时区（如：Asia/Shanghai）
	Packages    []string            `yaml:"packages,omitempty"`     // 要安装的软件包列表
	RunCmd      []string            `yaml:"runcmd,omitempty"`       // 启动后执行的命令
	WriteFiles  []WriteFile         `yaml:"write_files,omitempty"`  // 要写入的文件列表

	// 可以添加更多 cloud-init 支持的字段
	SSHPwauth      *bool                 `yaml:"ssh_pwauth,omitempty"`    // 启用 SSH 密码认证
	ChPasswd       *ChPasswd             `yaml:"chpasswd,omitempty"`      // 修改用户密码
	Locale         string                `yaml:"locale,omitempty"`        // 系统语言环境
	KeyboardLayout string                `yaml:"keyboard,omitempty"`      // 键盘布局
	Bootcmd        []string              `yaml:"bootcmd,omitempty"`       // 启动时执行的命令
	FinalMessage   string                `yaml:"final_message,omitempty"` // 完成后显示的消息
	PowerState     *PowerState           `yaml:"power_state,omitempty"`   // 电源状态配置
	APTSources     map[string]*APTSource `yaml:"apt_sources,omitempty"`   // APT 软件源配置
	Mounts         [][]string            `yaml:"mounts,omitempty"`        // 挂载点配置
	SSHKeys        *SSHKeys              `yaml:"ssh_keys,omitempty"`      // SSH 主机密钥
}

// ChPasswd 密码修改配置
type ChPasswd struct {
	Expire bool     `yaml:"expire,omitempty"` // 首次登录时强制修改密码
	List   []string `yaml:"list,omitempty"`   // 用户：密码 列表
}

// PowerState 电源状态配置
type PowerState struct {
	Delay     string `yaml:"delay,omitempty"`     // 延迟时间
	Mode      string `yaml:"mode,omitempty"`      // 模式：reboot, poweroff, halt
	Message   string `yaml:"message,omitempty"`   // 显示的消息
	Timeout   int    `yaml:"timeout,omitempty"`   // 超时时间
	Condition string `yaml:"condition,omitempty"` // 条件表达式
}

// APTSource APT 软件源配置
type APTSource struct {
	Source string `yaml:"source"`        // 源地址
	Key    string `yaml:"key,omitempty"` // GPG 密钥
}

// SSHKeys SSH 主机密钥配置
type SSHKeys struct {
	RSAPrivate     string `yaml:"rsa_private,omitempty"`
	RSAPublic      string `yaml:"rsa_public,omitempty"`
	ECDSAPrivate   string `yaml:"ecdsa_private,omitempty"`
	ECDSAPublic    string `yaml:"ecdsa_public,omitempty"`
	ED25519Private string `yaml:"ed25519_private,omitempty"`
	ED25519Public  string `yaml:"ed25519_public,omitempty"`
}

// WriteFile cloud-init 写入文件配置（用于序列化）
type WriteFile struct {
	Path        string `yaml:"path"`
	Content     string `yaml:"content"`
	Owner       string `yaml:"owner,omitempty"`
	Permissions string `yaml:"permissions,omitempty"`
	Encoding    string `yaml:"encoding,omitempty"` // 编码：base64, gzip, gz+base64
	Append      bool   `yaml:"append,omitempty"`   // 追加模式
	Defer       bool   `yaml:"defer,omitempty"`    // 延迟写入
}

// NetworkData 标准的 cloud-init network-config 结构
// 可直接序列化为 YAML 格式
type NetworkData struct {
	Version   string              `yaml:"version"`
	Ethernets map[string]Ethernet `yaml:"ethernets,omitempty"`
	Bonds     map[string]Bond     `yaml:"bonds,omitempty"`   // 网卡绑定
	Bridges   map[string]Bridge   `yaml:"bridges,omitempty"` // 网桥
	VLANs     map[string]VLAN     `yaml:"vlans,omitempty"`   // VLAN
}

// Bond 网卡绑定配置
type Bond struct {
	Interfaces  []string          `yaml:"interfaces"`
	Parameters  map[string]string `yaml:"parameters,omitempty"`
	Addresses   []string          `yaml:"addresses,omitempty"`
	Gateway4    string            `yaml:"gateway4,omitempty"`
	Nameservers struct {
		Addresses []string `yaml:"addresses,omitempty"`
	} `yaml:"nameservers,omitempty"`
}

// Bridge 网桥配置
type Bridge struct {
	Interfaces  []string `yaml:"interfaces"`
	DHCP4       bool     `yaml:"dhcp4,omitempty"`
	Addresses   []string `yaml:"addresses,omitempty"`
	Gateway4    string   `yaml:"gateway4,omitempty"`
	Nameservers struct {
		Addresses []string `yaml:"addresses,omitempty"`
	} `yaml:"nameservers,omitempty"`
}

// VLAN 配置
type VLAN struct {
	ID        int      `yaml:"id"`
	Link      string   `yaml:"link"`
	DHCP4     bool     `yaml:"dhcp4,omitempty"`
	Addresses []string `yaml:"addresses,omitempty"`
	Gateway4  string   `yaml:"gateway4,omitempty"`
}
