package libvirt

import (
	"encoding/xml"
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

// AttachDiskToDomain 附加磁盘到 domain
func (c *Client) AttachDiskToDomain(domainName, volumePath, device string) error {
	// 查找 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return fmt.Errorf("lookup domain: %w", err)
	}

	// 获取当前 domain XML
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 XML
	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		return fmt.Errorf("unmarshal domain XML: %w", err)
	}

	// 检查设备是否已存在
	for _, disk := range domainXML.Devices.Disks {
		if disk.Target.Dev == device {
			return fmt.Errorf("device %s already exists in domain", device)
		}
	}

	// 添加新磁盘
	newDisk := DomainDisk{
		Type:   "file",
		Device: "disk",
		Driver: DomainDiskDriver{
			Name: "qemu",
			Type: "qcow2",
		},
		Source: DomainDiskSource{
			File: volumePath,
		},
		Target: DomainDiskTarget{
			Dev: device,
			Bus: "virtio",
		},
	}

	domainXML.Devices.Disks = append(domainXML.Devices.Disks, newDisk)

	// 重新序列化 XML
	xmlBytes, err := xml.MarshalIndent(&domainXML, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal domain XML: %w", err)
	}

	// 更新 domain 定义
	_, err = c.conn.DomainDefineXML(string(xmlBytes))
	if err != nil {
		return fmt.Errorf("define domain with new disk: %w", err)
	}

	// 如果 domain 正在运行，需要热插拔磁盘
	state, _, err := c.conn.DomainGetState(domain, 0)
	if err != nil {
		return fmt.Errorf("get domain state: %w", err)
	}

	if libvirt.DomainState(state) == libvirt.DomainRunning {
		// 使用 AttachDeviceFlags 进行热插拔
		diskXML := fmt.Sprintf(`<disk type="file" device="disk">
  <driver name="qemu" type="qcow2"/>
  <source file="%s"/>
  <target dev="%s" bus="virtio"/>
</disk>`, volumePath, device)

		err = c.conn.DomainAttachDeviceFlags(domain, diskXML, uint32(libvirt.DomainDeviceModifyLive|libvirt.DomainDeviceModifyConfig))
		if err != nil {
			return fmt.Errorf("attach device to running domain: %w", err)
		}
	}

	return nil
}

// DetachDiskFromDomain 从 domain 分离磁盘
func (c *Client) DetachDiskFromDomain(domainName, device string) error {
	// 查找 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return fmt.Errorf("lookup domain: %w", err)
	}

	// 获取当前 domain XML
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 XML
	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		return fmt.Errorf("unmarshal domain XML: %w", err)
	}

	// 查找并删除磁盘
	found := false
	newDisks := make([]DomainDisk, 0, len(domainXML.Devices.Disks))
	for _, disk := range domainXML.Devices.Disks {
		if disk.Target.Dev == device {
			found = true
			continue
		}
		newDisks = append(newDisks, disk)
	}

	if !found {
		return fmt.Errorf("device %s not found in domain", device)
	}

	domainXML.Devices.Disks = newDisks

	// 如果 domain 正在运行，需要热拔磁盘
	state, _, err := c.conn.DomainGetState(domain, 0)
	if err != nil {
		return fmt.Errorf("get domain state: %w", err)
	}

	if libvirt.DomainState(state) == libvirt.DomainRunning {
		// 构建要分离的磁盘 XML（只需要关键字段）
		diskXML := fmt.Sprintf(`<disk>
  <target dev="%s" bus="virtio"/>
</disk>`, device)

		err = c.conn.DomainDetachDeviceFlags(domain, diskXML, uint32(libvirt.DomainDeviceModifyLive|libvirt.DomainDeviceModifyConfig))
		if err != nil {
			return fmt.Errorf("detach device from running domain: %w", err)
		}
	}

	// 重新序列化 XML
	xmlBytes, err := xml.MarshalIndent(&domainXML, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal domain XML: %w", err)
	}

	// 更新 domain 定义
	_, err = c.conn.DomainDefineXML(string(xmlBytes))
	if err != nil {
		return fmt.Errorf("define domain without disk: %w", err)
	}

	return nil
}

// GetDomainDisks 获取 domain 的所有磁盘设备
func (c *Client) GetDomainDisks(domainName string) ([]DomainDisk, error) {
	// 查找 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("lookup domain: %w", err)
	}

	// 获取 domain XML
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 XML
	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		return nil, fmt.Errorf("unmarshal domain XML: %w", err)
	}

	return domainXML.Devices.Disks, nil
}
