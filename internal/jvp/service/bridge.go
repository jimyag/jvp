package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jimyag/jvp/internal/jvp/entity"
)

// BridgeService 宿主机网桥服务
// 管理宿主机上的网桥设备
type BridgeService struct {
	nodeStorage *NodeStorage
}

// NewBridgeService 创建网桥服务
func NewBridgeService(nodeStorage *NodeStorage) *BridgeService {
	return &BridgeService{
		nodeStorage: nodeStorage,
	}
}

// ipLinkOutput ip -j link show 的输出结构
type ipLinkOutput struct {
	IfIndex   int      `json:"ifindex"`
	IfName    string   `json:"ifname"`
	Flags     []string `json:"flags"`
	MTU       int      `json:"mtu"`
	QDisc     string   `json:"qdisc"`
	Operstate string   `json:"operstate"`
	LinkType  string   `json:"linktype"`
	Address   string   `json:"address"`
	Broadcast string   `json:"broadcast"`
	Master    string   `json:"master,omitempty"` // 所属的 master 设备名（如网桥名）
	AddrInfo  []struct {
		Family    string `json:"family"`
		Local     string `json:"local"`
		Prefixlen int    `json:"prefixlen"`
		Scope     string `json:"scope"`
	} `json:"addr_info"`
}

// executeCommand 在指定节点执行命令
func (s *BridgeService) executeCommand(ctx context.Context, nodeName, command string) ([]byte, error) {
	if nodeName == "" || nodeName == "local" {
		// 本地执行
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		return cmd.Output()
	}

	// 远程节点：通过 SSH 执行
	client, err := s.nodeStorage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get connection for node %s: %w", nodeName, err)
	}

	// 获取 SSH 目标
	target, err := client.GetSSHTarget()
	if err != nil {
		return nil, fmt.Errorf("get SSH target: %w", err)
	}

	// 通过 SSH 执行命令
	cmd := exec.CommandContext(ctx, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes", target, command)
	return cmd.Output()
}

// ListBridges 列举宿主机上的网桥
func (s *BridgeService) ListBridges(ctx context.Context, nodeName string) ([]entity.HostBridge, error) {
	// 一次性获取所有网络接口（包含 master 信息）
	allLinksOutput, err := s.executeCommand(ctx, nodeName, "ip -j link show")
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}

	var allLinks []ipLinkOutput
	if err := json.Unmarshal(allLinksOutput, &allLinks); err != nil {
		return nil, fmt.Errorf("parse network interfaces: %w", err)
	}

	// 构建网桥名到绑定接口的映射
	bridgeInterfaces := make(map[string][]string)
	bridgeLinks := make([]ipLinkOutput, 0)

	for _, link := range allLinks {
		// 如果接口有 master，记录到对应网桥的绑定列表
		if link.Master != "" {
			bridgeInterfaces[link.Master] = append(bridgeInterfaces[link.Master], link.IfName)
		}
	}

	// 获取网桥列表
	bridgeOutput, err := s.executeCommand(ctx, nodeName, "ip -j link show type bridge")
	if err != nil {
		return nil, fmt.Errorf("list bridges: %w", err)
	}

	if err := json.Unmarshal(bridgeOutput, &bridgeLinks); err != nil {
		return nil, fmt.Errorf("parse bridge list: %w", err)
	}

	bridges := make([]entity.HostBridge, 0, len(bridgeLinks))
	for _, link := range bridgeLinks {
		// 获取桥接设备的 IP 地址
		ips := make([]string, 0)
		ipOutput, err := s.executeCommand(ctx, nodeName, fmt.Sprintf("ip -j addr show %s", link.IfName))
		if err == nil {
			var addrLinks []ipLinkOutput
			if json.Unmarshal(ipOutput, &addrLinks) == nil && len(addrLinks) > 0 {
				for _, addr := range addrLinks[0].AddrInfo {
					if addr.Family == "inet" {
						ips = append(ips, fmt.Sprintf("%s/%d", addr.Local, addr.Prefixlen))
					}
				}
			}
		}

		// 从预先构建的映射中获取绑定的接口
		interfaces := bridgeInterfaces[link.IfName]
		if interfaces == nil {
			interfaces = []string{}
		}

		// 检查 STP 状态
		stp := false
		stpOutput, err := s.executeCommand(ctx, nodeName, fmt.Sprintf("cat /sys/class/net/%s/bridge/stp_state 2>/dev/null || echo 0", link.IfName))
		if err == nil {
			stp = strings.TrimSpace(string(stpOutput)) != "0"
		}

		state := "down"
		if link.Operstate == "up" || link.Operstate == "UP" || containsFlag(link.Flags, "UP") {
			state = "up"
		}

		bridges = append(bridges, entity.HostBridge{
			Name:       link.IfName,
			State:      state,
			MAC:        link.Address,
			IPs:        ips,
			Interfaces: interfaces,
			STP:        stp,
			MTU:        link.MTU,
		})
	}

	return bridges, nil
}

// CreateBridge 创建网桥
func (s *BridgeService) CreateBridge(ctx context.Context, req *entity.CreateBridgeRequest) (*entity.HostBridge, error) {
	// 创建网桥
	cmd := fmt.Sprintf("ip link add %s type bridge", req.BridgeName)
	if _, err := s.executeCommand(ctx, req.NodeName, cmd); err != nil {
		return nil, fmt.Errorf("create bridge: %w", err)
	}

	// 设置 STP
	stpVal := "0"
	if req.STP {
		stpVal = "1"
	}
	cmd = fmt.Sprintf("echo %s > /sys/class/net/%s/bridge/stp_state", stpVal, req.BridgeName)
	_, _ = s.executeCommand(ctx, req.NodeName, cmd)

	// 启用网桥
	cmd = fmt.Sprintf("ip link set %s up", req.BridgeName)
	if _, err := s.executeCommand(ctx, req.NodeName, cmd); err != nil {
		// 启用失败，删除已创建的网桥
		_, _ = s.executeCommand(ctx, req.NodeName, fmt.Sprintf("ip link del %s", req.BridgeName))
		return nil, fmt.Errorf("enable bridge: %w", err)
	}

	// 绑定指定的网络接口
	for _, iface := range req.Interfaces {
		// 将接口添加到网桥
		cmd = fmt.Sprintf("ip link set %s master %s", iface, req.BridgeName)
		if _, err := s.executeCommand(ctx, req.NodeName, cmd); err != nil {
			// 绑定失败，继续处理其他接口，但记录错误
			continue
		}
		// 确保接口是启用状态
		cmd = fmt.Sprintf("ip link set %s up", iface)
		_, _ = s.executeCommand(ctx, req.NodeName, cmd)
	}

	// 获取创建后的网桥信息
	bridges, err := s.ListBridges(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get bridge info: %w", err)
	}

	for _, br := range bridges {
		if br.Name == req.BridgeName {
			return &br, nil
		}
	}

	return &entity.HostBridge{
		Name:  req.BridgeName,
		State: "up",
		STP:   req.STP,
	}, nil
}

// ListAvailableInterfaces 列举可用于绑定到网桥的网络接口
func (s *BridgeService) ListAvailableInterfaces(ctx context.Context, nodeName string) ([]entity.NetworkInterface, error) {
	// 获取所有网络接口
	allLinksOutput, err := s.executeCommand(ctx, nodeName, "ip -j link show")
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}

	var allLinks []ipLinkOutput
	if err := json.Unmarshal(allLinksOutput, &allLinks); err != nil {
		return nil, fmt.Errorf("parse network interfaces: %w", err)
	}

	// 获取所有物理网卡列表（通过检查 /sys/class/net/*/device 是否存在）
	// 物理设备会有 device 符号链接指向 PCI 设备，虚拟设备没有
	physicalNics := make(map[string]bool)
	physicalOutput, err := s.executeCommand(ctx, nodeName, "for iface in $(ls /sys/class/net/); do [ -e \"/sys/class/net/$iface/device\" ] && echo \"$iface\"; done")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(physicalOutput)), "\n")
		for _, line := range lines {
			if name := strings.TrimSpace(line); name != "" {
				physicalNics[name] = true
			}
		}
	}

	interfaces := make([]entity.NetworkInterface, 0)
	for _, link := range allLinks {
		// 跳过 loopback
		if link.IfName == "lo" {
			continue
		}
		// 跳过 bridge 类型
		if link.LinkType == "bridge" {
			continue
		}

		// 判断是否是物理网卡
		isPhysical := physicalNics[link.IfName]

		// 如果不是物理网卡，跳过
		if !isPhysical {
			continue
		}

		state := "down"
		if link.Operstate == "up" || link.Operstate == "UP" || containsFlag(link.Flags, "UP") {
			state = "up"
		}

		interfaces = append(interfaces, entity.NetworkInterface{
			Name:    link.IfName,
			MAC:     link.Address,
			State:   state,
			BoundTo: link.Master, // 记录绑定到的网桥（空表示未绑定）
		})
	}

	return interfaces, nil
}

// DeleteBridge 删除网桥
func (s *BridgeService) DeleteBridge(ctx context.Context, nodeName, bridgeName string) error {
	// 先停用网桥
	cmd := fmt.Sprintf("ip link set %s down", bridgeName)
	_, _ = s.executeCommand(ctx, nodeName, cmd)

	// 删除网桥
	cmd = fmt.Sprintf("ip link del %s", bridgeName)
	if _, err := s.executeCommand(ctx, nodeName, cmd); err != nil {
		return fmt.Errorf("delete bridge %s: %w", bridgeName, err)
	}

	return nil
}

// containsFlag 检查 flags 中是否包含指定标志
func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}
