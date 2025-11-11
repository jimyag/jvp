// Package entity 定义业务实体
package entity

// KeyPair 密钥对信息
type KeyPair struct {
	ID          string `json:"id"`          // 密钥对 ID: kp-{uuid}
	Name        string `json:"name"`        // 密钥对名称
	Algorithm   string `json:"algorithm"`   // 算法：rsa, ed25519
	PublicKey   string `json:"public_key"`  // 公钥内容
	Fingerprint string `json:"fingerprint"` // 公钥指纹（SHA256）
	CreatedAt   string `json:"created_at"`  // 创建时间
	// 注意：私钥不存储在数据库中，只在创建时返回一次
}

// CreateKeyPairRequest 创建密钥对请求
type CreateKeyPairRequest struct {
	Name      string `json:"name" binding:"required"` // 密钥对名称
	Algorithm string `json:"algorithm"`               // 算法：rsa, ed25519（默认：ed25519）
	KeySize   int    `json:"key_size,omitempty"`      // RSA 密钥长度（默认：2048，仅 RSA 使用）
}

// CreateKeyPairResponse 创建密钥对响应
type CreateKeyPairResponse struct {
	KeyPair    *KeyPair `json:"keypair"`     // 密钥对信息
	PrivateKey string   `json:"private_key"` // 私钥（仅返回一次）
}

// ImportKeyPairRequest 导入密钥对请求
type ImportKeyPairRequest struct {
	Name      string `json:"name" binding:"required"`       // 密钥对名称
	PublicKey string `json:"public_key" binding:"required"` // 公钥内容
}

// ImportKeyPairResponse 导入密钥对响应
type ImportKeyPairResponse struct {
	KeyPair *KeyPair `json:"keypair"`
}

// DeleteKeyPairRequest 删除密钥对请求
type DeleteKeyPairRequest struct {
	KeyPairID string `json:"keypairID" binding:"required"`
}

// DeleteKeyPairResponse 删除密钥对响应
type DeleteKeyPairResponse struct {
	Return bool `json:"return"`
}

// DescribeKeyPairsRequest 描述密钥对请求
type DescribeKeyPairsRequest struct {
	KeyPairIDs []string `json:"keypairIDs,omitempty"` // 密钥对 ID 列表
	Filters    []Filter `json:"filters,omitempty"`    // 过滤器
	MaxResults int      `json:"maxResults,omitempty"` // 最大结果数
	NextToken  string   `json:"nextToken,omitempty"`  // 分页令牌
}

// DescribeKeyPairsResponse 描述密钥对响应
type DescribeKeyPairsResponse struct {
	KeyPairs  []KeyPair `json:"keypairs"`
	NextToken string    `json:"nextToken,omitempty"`
}
