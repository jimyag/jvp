package entity

// Network libvirt 虚拟网络实体
type Network struct {
	Name       string `json:"name"`                 // 网络名称
	UUID       string `json:"uuid,omitempty"`       // UUID
	NodeName   string `json:"node_name"`            // 所属节点
	Type       string `json:"type"`                 // 类型：libvirt/bridge
	Mode       string `json:"mode"`                 // 模式：nat/bridge/isolated/route
	Bridge     string `json:"bridge"`               // 关联的网桥名称
	State      string `json:"state"`                // 状态：active/inactive
	Autostart  bool   `json:"autostart"`            // 是否自动启动
	Persistent bool   `json:"persistent"`           // 是否持久化
	IPAddress  string `json:"ip_address,omitempty"` // 网关 IP
	Netmask    string `json:"netmask,omitempty"`    // 子网掩码
	DHCPStart  string `json:"dhcp_start,omitempty"` // DHCP 起始 IP
	DHCPEnd    string `json:"dhcp_end,omitempty"`   // DHCP 结束 IP
}

// HostBridge 宿主机网桥实体
type HostBridge struct {
	Name       string   `json:"name"`                 // 网桥名称
	State      string   `json:"state"`                // 状态：up/down
	MAC        string   `json:"mac,omitempty"`        // MAC 地址
	IPs        []string `json:"ips,omitempty"`        // IP 地址列表
	Interfaces []string `json:"interfaces,omitempty"` // 绑定的物理接口
	STP        bool     `json:"stp"`                  // 是否启用 STP
	MTU        int      `json:"mtu,omitempty"`        // MTU
}

// NetworkSources 可用网络源（用于创建 VM 时选择）
type NetworkSources struct {
	LibvirtNetworks []Network    `json:"libvirt_networks"` // libvirt 虚拟网络
	HostBridges     []HostBridge `json:"host_bridges"`     // 宿主机网桥
}

// ============================================================================
// Network API 请求和响应
// ============================================================================

// ListNetworksRequest 列举网络请求
type ListNetworksRequest struct {
	NodeName string `json:"node_name" binding:"required"` // 节点名称
}

// ListNetworksResponse 列举网络响应
type ListNetworksResponse struct {
	Networks []Network `json:"networks"`
}

// DescribeNetworkRequest 查询网络详情请求
type DescribeNetworkRequest struct {
	NodeName    string `json:"node_name" binding:"required"`    // 节点名称
	NetworkName string `json:"network_name" binding:"required"` // 网络名称
}

// DescribeNetworkResponse 查询网络详情响应
type DescribeNetworkResponse struct {
	Network *Network `json:"network"`
}

// CreateNetworkRequest 创建网络请求
type CreateNetworkRequest struct {
	NodeName  string `json:"node_name" binding:"required"` // 节点名称
	Name      string `json:"name" binding:"required"`      // 网络名称
	Mode      string `json:"mode"`                         // 模式：nat/isolated（默认 nat）
	IPAddress string `json:"ip_address"`                   // 网关 IP（如 192.168.100.1）
	Netmask   string `json:"netmask"`                      // 子网掩码（如 255.255.255.0）
	DHCPStart string `json:"dhcp_start"`                   // DHCP 起始 IP
	DHCPEnd   string `json:"dhcp_end"`                     // DHCP 结束 IP
	Autostart bool   `json:"autostart"`                    // 是否自动启动
}

// CreateNetworkResponse 创建网络响应
type CreateNetworkResponse struct {
	Network *Network `json:"network"`
}

// DeleteNetworkRequest 删除网络请求
type DeleteNetworkRequest struct {
	NodeName    string `json:"node_name" binding:"required"`    // 节点名称
	NetworkName string `json:"network_name" binding:"required"` // 网络名称
}

// DeleteNetworkResponse 删除网络响应
type DeleteNetworkResponse struct {
	Message string `json:"message"`
}

// StartNetworkRequest 启动网络请求
type StartNetworkRequest struct {
	NodeName    string `json:"node_name" binding:"required"`    // 节点名称
	NetworkName string `json:"network_name" binding:"required"` // 网络名称
}

// StartNetworkResponse 启动网络响应
type StartNetworkResponse struct {
	Network *Network `json:"network"`
}

// StopNetworkRequest 停止网络请求
type StopNetworkRequest struct {
	NodeName    string `json:"node_name" binding:"required"`    // 节点名称
	NetworkName string `json:"network_name" binding:"required"` // 网络名称
}

// StopNetworkResponse 停止网络响应
type StopNetworkResponse struct {
	Network *Network `json:"network"`
}

// ListNetworkSourcesRequest 列举可用网络源请求
type ListNetworkSourcesRequest struct {
	NodeName string `json:"node_name" binding:"required"` // 节点名称
}

// ListNetworkSourcesResponse 列举可用网络源响应
type ListNetworkSourcesResponse struct {
	Sources *NetworkSources `json:"sources"`
}

// ============================================================================
// Bridge API 请求和响应
// ============================================================================

// ListBridgesRequest 列举网桥请求
type ListBridgesRequest struct {
	NodeName string `json:"node_name" binding:"required"` // 节点名称
}

// ListBridgesResponse 列举网桥响应
type ListBridgesResponse struct {
	Bridges []HostBridge `json:"bridges"`
}

// CreateBridgeRequest 创建网桥请求
type CreateBridgeRequest struct {
	NodeName   string   `json:"node_name" binding:"required"`   // 节点名称
	BridgeName string   `json:"bridge_name" binding:"required"` // 网桥名称
	STP        bool     `json:"stp"`                            // 是否启用 STP（默认 false）
	Interfaces []string `json:"interfaces,omitempty"`           // 要绑定的网络接口（可选）
}

// CreateBridgeResponse 创建网桥响应
type CreateBridgeResponse struct {
	Bridge *HostBridge `json:"bridge"`
}

// DeleteBridgeRequest 删除网桥请求
type DeleteBridgeRequest struct {
	NodeName   string `json:"node_name" binding:"required"`   // 节点名称
	BridgeName string `json:"bridge_name" binding:"required"` // 网桥名称
}

// DeleteBridgeResponse 删除网桥响应
type DeleteBridgeResponse struct {
	Message string `json:"message"`
}

// ListAvailableInterfacesRequest 列举可用网络接口请求
type ListAvailableInterfacesRequest struct {
	NodeName string `json:"node_name" binding:"required"` // 节点名称
}

// ListAvailableInterfacesResponse 列举可用网络接口响应
type ListAvailableInterfacesResponse struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}
