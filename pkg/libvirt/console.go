package libvirt

import (
	"encoding/xml"
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

// ConsoleInfo 控制台连接信息
type ConsoleInfo struct {
	VNCSocket    string `json:"vnc_socket"`     // VNC Unix Socket 路径
	SerialDevice string `json:"serial_device"`  // Serial PTY 设备路径
	Type         string `json:"type"`           // 控制台类型: vnc, serial
}

// GetDomainConsoleInfo 获取 Domain 的控制台连接信息
func (c *Client) GetDomainConsoleInfo(domain libvirt.Domain) (*ConsoleInfo, error) {
	// 获取 Domain XML
	xmlData, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %w", err)
	}

	// 解析 XML
	var domainDef DomainXML
	if err := xml.Unmarshal([]byte(xmlData), &domainDef); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	info := &ConsoleInfo{}

	// 获取 VNC Socket 路径
	if domainDef.Devices.Graphics.Type == "vnc" && domainDef.Devices.Graphics.Socket != "" {
		info.VNCSocket = domainDef.Devices.Graphics.Socket
		info.Type = "vnc"
	}

	// 获取 Serial Console PTY 设备路径
	// Serial Console 的 PTY 路径在运行时由 libvirt 分配,需要从运行时 XML 中获取
	if domainDef.Devices.Console.Type == "pty" {
		// 重新获取运行时 XML (flags=0 表示运行时状态)
		runtimeXML, err := c.conn.DomainGetXMLDesc(domain, 0)
		if err == nil {
			var runtimeDef DomainXML
			if err := xml.Unmarshal([]byte(runtimeXML), &runtimeDef); err == nil {
				// PTY 设备路径在 console source path 中
				if runtimeDef.Devices.Console.Source.Path != "" {
					info.SerialDevice = runtimeDef.Devices.Console.Source.Path
				}
			}
		}
	}

	return info, nil
}
