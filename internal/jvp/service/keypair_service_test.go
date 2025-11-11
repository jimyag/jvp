package service

import (
	"context"
	"strings"
	"testing"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyPairService_CreateKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		req           *entity.CreateKeyPairRequest
		expectError   bool
		errorContains string
		validate      func(*testing.T, *entity.CreateKeyPairResponse)
	}{
		{
			name: "create ed25519 keypair",
			req: &entity.CreateKeyPairRequest{
				Name:      "test-keypair",
				Algorithm: "ed25519",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.CreateKeyPairResponse) {
				assert.NotNil(t, resp.KeyPair)
				assert.NotEmpty(t, resp.KeyPair.ID)
				assert.True(t, strings.HasPrefix(resp.KeyPair.ID, "kp-"))
				assert.Equal(t, "test-keypair", resp.KeyPair.Name)
				assert.Equal(t, "ed25519", resp.KeyPair.Algorithm)
				assert.NotEmpty(t, resp.KeyPair.PublicKey)
				assert.True(t, strings.HasPrefix(resp.KeyPair.PublicKey, "ssh-ed25519"))
				assert.NotEmpty(t, resp.KeyPair.Fingerprint)
				assert.True(t, strings.HasPrefix(resp.KeyPair.Fingerprint, "SHA256:"))
				assert.NotEmpty(t, resp.PrivateKey)
				assert.True(t, strings.Contains(resp.PrivateKey, "PRIVATE KEY") || strings.Contains(resp.PrivateKey, "BEGIN"))
			},
		},
		{
			name: "create rsa keypair",
			req: &entity.CreateKeyPairRequest{
				Name:      "test-rsa-keypair",
				Algorithm: "rsa",
				KeySize:   2048,
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.CreateKeyPairResponse) {
				assert.NotNil(t, resp.KeyPair)
				assert.NotEmpty(t, resp.KeyPair.ID)
				assert.Equal(t, "test-rsa-keypair", resp.KeyPair.Name)
				assert.Equal(t, "rsa", resp.KeyPair.Algorithm)
				assert.NotEmpty(t, resp.KeyPair.PublicKey)
				assert.True(t, strings.HasPrefix(resp.KeyPair.PublicKey, "ssh-rsa"))
				assert.NotEmpty(t, resp.KeyPair.Fingerprint)
				assert.NotEmpty(t, resp.PrivateKey)
			},
		},
		{
			name: "create keypair with default algorithm",
			req: &entity.CreateKeyPairRequest{
				Name: "default-algorithm",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.CreateKeyPairResponse) {
				assert.Equal(t, "ed25519", resp.KeyPair.Algorithm)
			},
		},
		{
			name: "create keypair with invalid algorithm",
			req: &entity.CreateKeyPairRequest{
				Name:      "invalid-algorithm",
				Algorithm: "ecdsa",
			},
			expectError:   true,
			errorContains: "unsupported algorithm",
		},
		{
			name: "create rsa keypair with invalid key size",
			req: &entity.CreateKeyPairRequest{
				Name:      "invalid-rsa-size",
				Algorithm: "rsa",
				KeySize:   1024,
			},
			expectError:   true,
			errorContains: "RSA key size must be at least 2048",
		},
		{
			name: "create rsa keypair with default key size",
			req: &entity.CreateKeyPairRequest{
				Name:      "default-rsa-size",
				Algorithm: "rsa",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.CreateKeyPairResponse) {
				assert.Equal(t, "rsa", resp.KeyPair.Algorithm)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			services := setupTestServices(t)
			ctx := context.Background()

			resp, err := services.KeyPairService.CreateKeyPair(ctx, tc.req)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.validate != nil {
				tc.validate(t, resp)
			}
		})
	}
}

func TestKeyPairService_ImportKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name           string
		setupPublicKey func(*KeyPairService, context.Context) string // 返回公钥字符串
		req            *entity.ImportKeyPairRequest
		expectError    bool
		errorContains  string
		validate       func(*testing.T, *entity.ImportKeyPairResponse)
	}{
		{
			name: "import ed25519 public key",
			setupPublicKey: func(s *KeyPairService, ctx context.Context) string {
				resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
					Name:      "temp-ed25519",
					Algorithm: "ed25519",
				})
				if err != nil {
					panic(err)
				}
				return resp.KeyPair.PublicKey
			},
			req: &entity.ImportKeyPairRequest{
				Name: "imported-ed25519",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ImportKeyPairResponse) {
				assert.NotNil(t, resp.KeyPair)
				assert.NotEmpty(t, resp.KeyPair.ID)
				assert.Equal(t, "imported-ed25519", resp.KeyPair.Name)
				assert.Equal(t, "ed25519", resp.KeyPair.Algorithm)
				assert.NotEmpty(t, resp.KeyPair.PublicKey)
				assert.NotEmpty(t, resp.KeyPair.Fingerprint)
			},
		},
		{
			name: "import rsa public key",
			setupPublicKey: func(s *KeyPairService, ctx context.Context) string {
				resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
					Name:      "temp-rsa",
					Algorithm: "rsa",
					KeySize:   2048,
				})
				if err != nil {
					panic(err)
				}
				return resp.KeyPair.PublicKey
			},
			req: &entity.ImportKeyPairRequest{
				Name: "imported-rsa",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ImportKeyPairResponse) {
				assert.NotNil(t, resp.KeyPair)
				assert.Equal(t, "imported-rsa", resp.KeyPair.Name)
				assert.Equal(t, "rsa", resp.KeyPair.Algorithm)
			},
		},
		{
			name:           "import invalid public key format",
			setupPublicKey: nil,
			req: &entity.ImportKeyPairRequest{
				Name:      "invalid-key",
				PublicKey: "invalid-key-format",
			},
			expectError:   true,
			errorContains: "invalid public key format",
		},
		{
			name:           "import empty public key",
			setupPublicKey: nil,
			req: &entity.ImportKeyPairRequest{
				Name:      "empty-key",
				PublicKey: "",
			},
			expectError:   true,
			errorContains: "invalid public key format",
		},
		{
			name: "import duplicate name",
			setupPublicKey: func(s *KeyPairService, ctx context.Context) string {
				resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
					Name:      "temp-for-duplicate",
					Algorithm: "ed25519",
				})
				if err != nil {
					panic(err)
				}
				return resp.KeyPair.PublicKey
			},
			req: &entity.ImportKeyPairRequest{
				Name: "duplicate-name",
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ImportKeyPairResponse) {
				// 导入第二个相同名称的密钥对（应该允许）
				testServices := setupTestServices(t)
				testCtx := context.Background()
				// 生成一个新的公钥用于第二个导入
				tempResp, err := testServices.KeyPairService.CreateKeyPair(testCtx, &entity.CreateKeyPairRequest{
					Name:      "temp-for-duplicate-2",
					Algorithm: "ed25519",
				})
				require.NoError(t, err)
				resp2, err := testServices.KeyPairService.ImportKeyPair(testCtx, &entity.ImportKeyPairRequest{
					Name:      "duplicate-name",
					PublicKey: tempResp.KeyPair.PublicKey,
				})
				require.NoError(t, err)
				assert.NotEqual(t, resp.KeyPair.ID, resp2.KeyPair.ID)
				assert.Equal(t, resp.KeyPair.Name, resp2.KeyPair.Name)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testServices := setupTestServices(t)
			testCtx := context.Background()

			// 如果测试用例需要有效的公钥，先创建
			req := tc.req
			if tc.setupPublicKey != nil {
				publicKey := tc.setupPublicKey(testServices.KeyPairService, testCtx)
				req = &entity.ImportKeyPairRequest{
					Name:      req.Name,
					PublicKey: publicKey,
				}
			}

			resp, err := testServices.KeyPairService.ImportKeyPair(testCtx, req)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.validate != nil {
				tc.validate(t, resp)
			}
		})
	}
}

func TestKeyPairService_DeleteKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		setupKeyPair  func(*KeyPairService) string
		keyPairID     string
		expectError   bool
		errorContains string
	}{
		{
			name: "delete existing keypair",
			setupKeyPair: func(s *KeyPairService) string {
				ctx := context.Background()
				resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
					Name:      "to-delete",
					Algorithm: "ed25519",
				})
				require.NoError(t, err)
				return resp.KeyPair.ID
			},
			expectError: false,
		},
		{
			name: "delete non-existent keypair",
			setupKeyPair: func(s *KeyPairService) string {
				return "kp-nonexistent"
			},
			keyPairID:     "kp-nonexistent",
			expectError:   true,
			errorContains: "not found",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			services := setupTestServices(t)
			ctx := context.Background()

			keyPairID := tc.keyPairID
			if tc.setupKeyPair != nil {
				keyPairID = tc.setupKeyPair(services.KeyPairService)
			}

			err := services.KeyPairService.DeleteKeyPair(ctx, keyPairID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// 验证密钥对已被删除（软删除）
			_, err = services.KeyPairService.GetKeyPairByID(ctx, keyPairID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	}
}

func TestKeyPairService_DescribeKeyPairs(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		setupKeyPairs func(*KeyPairService) []string
		req           *entity.DescribeKeyPairsRequest
		expectCount   int
		validate      func(*testing.T, []entity.KeyPair, []string)
	}{
		{
			name: "describe all keypairs",
			setupKeyPairs: func(s *KeyPairService) []string {
				ctx := context.Background()
				var ids []string
				for i := 0; i < 3; i++ {
					resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
						Name:      "keypair-" + string(rune('a'+i)),
						Algorithm: "ed25519",
					})
					if err != nil {
						return nil
					}
					ids = append(ids, resp.KeyPair.ID)
				}
				return ids
			},
			req:         &entity.DescribeKeyPairsRequest{},
			expectCount: 3,
			validate: func(t *testing.T, keyPairs []entity.KeyPair, ids []string) {
				assert.Len(t, keyPairs, 3)
				for _, kp := range keyPairs {
					assert.NotEmpty(t, kp.ID)
					assert.NotEmpty(t, kp.Name)
					assert.NotEmpty(t, kp.PublicKey)
					assert.NotEmpty(t, kp.Fingerprint)
				}
			},
		},
		{
			name: "describe by keypair ids",
			setupKeyPairs: func(s *KeyPairService) []string {
				ctx := context.Background()
				var ids []string
				for i := 0; i < 3; i++ {
					resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
						Name:      "keypair-" + string(rune('a'+i)),
						Algorithm: "ed25519",
					})
					if err != nil {
						return nil
					}
					ids = append(ids, resp.KeyPair.ID)
				}
				return ids
			},
			req: &entity.DescribeKeyPairsRequest{
				KeyPairIDs: []string{},
			},
			expectCount: 0,
		},
		{
			name: "describe by specific keypair ids",
			setupKeyPairs: func(s *KeyPairService) []string {
				ctx := context.Background()
				var ids []string
				for i := 0; i < 3; i++ {
					resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
						Name:      "keypair-" + string(rune('a'+i)),
						Algorithm: "ed25519",
					})
					if err != nil {
						return nil
					}
					ids = append(ids, resp.KeyPair.ID)
				}
				return ids
			},
			req: &entity.DescribeKeyPairsRequest{
				KeyPairIDs: []string{},
			},
			expectCount: 0,
			validate: func(t *testing.T, keyPairs []entity.KeyPair, ids []string) {
				// 验证返回的密钥对数量是 2 个
				assert.Len(t, keyPairs, 2)
				// 验证返回的密钥对 ID 都在请求的 IDs 中
				returnedIDs := make(map[string]bool)
				for _, kp := range keyPairs {
					returnedIDs[kp.ID] = true
				}
				// ids 应该至少有 2 个
				if len(ids) >= 2 {
					for _, id := range ids[:2] {
						assert.True(t, returnedIDs[id], "keypair %s should be in results", id)
					}
				}
			},
		},
		{
			name: "describe empty list",
			setupKeyPairs: func(s *KeyPairService) []string {
				return nil
			},
			req:         &entity.DescribeKeyPairsRequest{},
			expectCount: 0,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			services := setupTestServices(t)
			ctx := context.Background()

			var ids []string
			if tc.setupKeyPairs != nil {
				ids = tc.setupKeyPairs(services.KeyPairService)
			}

			// 如果测试用例是 "describe by specific keypair ids"，使用 setupKeyPairs 返回的 IDs
			if tc.name == "describe by specific keypair ids" {
				if len(ids) < 2 {
					t.Fatal("not enough keypairs created")
				}
				req := &entity.DescribeKeyPairsRequest{
					KeyPairIDs: ids[:2],
				}
				keyPairs, err := services.KeyPairService.DescribeKeyPairs(ctx, req)
				require.NoError(t, err)
				assert.Len(t, keyPairs, 2)
				if tc.validate != nil {
					tc.validate(t, keyPairs, ids)
				}
				return
			}

			keyPairs, err := services.KeyPairService.DescribeKeyPairs(ctx, tc.req)
			require.NoError(t, err)
			assert.Len(t, keyPairs, tc.expectCount)
			if tc.validate != nil {
				tc.validate(t, keyPairs, ids) // ids 可能为 nil，但 validate 函数应该能处理
			}
		})
	}
}

func TestKeyPairService_GetKeyPairByID(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		setupKeyPair  func(*KeyPairService) string
		keyPairID     string
		expectError   bool
		errorContains string
		validate      func(*testing.T, *entity.KeyPair)
	}{
		{
			name: "get existing keypair",
			setupKeyPair: func(s *KeyPairService) string {
				ctx := context.Background()
				resp, err := s.CreateKeyPair(ctx, &entity.CreateKeyPairRequest{
					Name:      "test-keypair",
					Algorithm: "ed25519",
				})
				require.NoError(t, err)
				return resp.KeyPair.ID
			},
			expectError: false,
			validate: func(t *testing.T, kp *entity.KeyPair) {
				assert.NotNil(t, kp)
				assert.Equal(t, "test-keypair", kp.Name)
				assert.Equal(t, "ed25519", kp.Algorithm)
				assert.NotEmpty(t, kp.PublicKey)
				assert.NotEmpty(t, kp.Fingerprint)
			},
		},
		{
			name: "get non-existent keypair",
			setupKeyPair: func(s *KeyPairService) string {
				return "kp-nonexistent"
			},
			keyPairID:     "kp-nonexistent",
			expectError:   true,
			errorContains: "not found",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			services := setupTestServices(t)
			ctx := context.Background()

			keyPairID := tc.keyPairID
			if tc.setupKeyPair != nil {
				keyPairID = tc.setupKeyPair(services.KeyPairService)
			}

			kp, err := services.KeyPairService.GetKeyPairByID(ctx, keyPairID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, kp)
			if tc.validate != nil {
				tc.validate(t, kp)
			}
		})
	}
}
