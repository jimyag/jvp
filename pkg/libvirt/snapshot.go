package libvirt

import (
	"fmt"
)

// ListSnapshots 列出域的所有快照名称
func (c *Client) ListSnapshots(domainName string) ([]string, error) {
	// 获取 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	// 使用 DomainListAllSnapshots 获取所有快照
	// NeedResults: 设置为足够大的数字以获取所有快照
	// Flags: 0 表示获取所有类型的快照
	snapshots, _, err := c.conn.DomainListAllSnapshots(domain, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("list snapshots for domain %s: %w", domainName, err)
	}

	// 提取快照名称
	names := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		// 获取快照的 XML 描述以提取名称
		xmlDesc, err := c.conn.DomainSnapshotGetXMLDesc(snapshot, 0)
		if err != nil {
			continue
		}
		name := extractSnapshotName(xmlDesc)
		if name != "" {
			names = append(names, name)
		}
	}

	return names, nil
}

// extractSnapshotName 从快照 XML 中提取名称
func extractSnapshotName(xmlDesc string) string {
	// 查找 <name> 标签
	nameStart := 0
	for i := 0; i < len(xmlDesc); i++ {
		if i+6 <= len(xmlDesc) && xmlDesc[i:i+6] == "<name>" {
			nameStart = i + 6
			break
		}
	}
	if nameStart == 0 {
		return ""
	}

	// 查找结束标签
	nameEnd := nameStart
	for i := nameStart; i < len(xmlDesc); i++ {
		if i+7 <= len(xmlDesc) && xmlDesc[i:i+7] == "</name>" {
			nameEnd = i
			break
		}
	}
	if nameEnd == nameStart {
		return ""
	}

	return xmlDesc[nameStart:nameEnd]
}
