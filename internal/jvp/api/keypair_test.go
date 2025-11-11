package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKeyPairService 是 KeyPairService 的 mock 实现
type MockKeyPairService struct {
	mock.Mock
}

func (m *MockKeyPairService) CreateKeyPair(ctx context.Context, req *entity.CreateKeyPairRequest) (*entity.CreateKeyPairResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.CreateKeyPairResponse), args.Error(1)
}

func (m *MockKeyPairService) ImportKeyPair(ctx context.Context, req *entity.ImportKeyPairRequest) (*entity.ImportKeyPairResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ImportKeyPairResponse), args.Error(1)
}

func (m *MockKeyPairService) DeleteKeyPair(ctx context.Context, keyPairID string) error {
	args := m.Called(ctx, keyPairID)
	return args.Error(0)
}

func (m *MockKeyPairService) DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]entity.KeyPair, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.KeyPair), args.Error(1)
}

func TestKeyPair_CreateKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.CreateKeyPairRequest
		mockSetup    func(*MockKeyPairService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful create",
			req: &entity.CreateKeyPairRequest{
				Name:      "test-keypair",
				Algorithm: "ed25519",
			},
			mockSetup: func(m *MockKeyPairService) {
				m.On("CreateKeyPair", mock.Anything, mock.AnythingOfType("*entity.CreateKeyPairRequest")).
					Return(&entity.CreateKeyPairResponse{
						KeyPair: &entity.KeyPair{
							ID:          "kp-123",
							Name:        "test-keypair",
							Algorithm:   "ed25519",
							PublicKey:   "ssh-ed25519 AAAA...",
							Fingerprint: "SHA256:xxxxx",
							CreatedAt:   "2024-01-01T00:00:00Z",
						},
						PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "create with error",
			req: &entity.CreateKeyPairRequest{
				Name:      "error-keypair",
				Algorithm: "invalid",
			},
			mockSetup: func(m *MockKeyPairService) {
				m.On("CreateKeyPair", mock.Anything, mock.AnythingOfType("*entity.CreateKeyPairRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockKeyPairService)
			tc.mockSetup(mockService)

			keypairAPI := &KeyPair{
				keyPairService: mockService,
			}

			reqBody, err := json.Marshal(tc.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/keypairs/create", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/keypairs/create", ginx.Adapt5(keypairAPI.CreateKeyPair))
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestKeyPair_ImportKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.ImportKeyPairRequest
		mockSetup    func(*MockKeyPairService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful import",
			req: &entity.ImportKeyPairRequest{
				Name:      "imported-keypair",
				PublicKey: "ssh-ed25519 AAAA...",
			},
			mockSetup: func(m *MockKeyPairService) {
				m.On("ImportKeyPair", mock.Anything, mock.AnythingOfType("*entity.ImportKeyPairRequest")).
					Return(&entity.ImportKeyPairResponse{
						KeyPair: &entity.KeyPair{
							ID:          "kp-456",
							Name:        "imported-keypair",
							Algorithm:   "ed25519",
							PublicKey:   "ssh-ed25519 AAAA...",
							Fingerprint: "SHA256:yyyyy",
							CreatedAt:   "2024-01-01T00:00:00Z",
						},
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockKeyPairService)
			tc.mockSetup(mockService)

			keypairAPI := &KeyPair{
				keyPairService: mockService,
			}

			reqBody, err := json.Marshal(tc.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/keypairs/import", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/keypairs/import", ginx.Adapt5(keypairAPI.ImportKeyPair))
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestKeyPair_DeleteKeyPair(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DeleteKeyPairRequest
		mockSetup    func(*MockKeyPairService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful delete",
			req: &entity.DeleteKeyPairRequest{
				KeyPairID: "kp-123",
			},
			mockSetup: func(m *MockKeyPairService) {
				m.On("DeleteKeyPair", mock.Anything, "kp-123").Return(nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "delete non-existent keypair",
			req: &entity.DeleteKeyPairRequest{
				KeyPairID: "kp-nonexistent",
			},
			mockSetup: func(m *MockKeyPairService) {
				m.On("DeleteKeyPair", mock.Anything, "kp-nonexistent").Return(assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockKeyPairService)
			tc.mockSetup(mockService)

			keypairAPI := &KeyPair{
				keyPairService: mockService,
			}

			reqBody, err := json.Marshal(tc.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/keypairs/delete", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/keypairs/delete", ginx.Adapt5(keypairAPI.DeleteKeyPair))
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestKeyPair_DescribeKeyPairs(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DescribeKeyPairsRequest
		mockSetup    func(*MockKeyPairService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful describe",
			req:  &entity.DescribeKeyPairsRequest{},
			mockSetup: func(m *MockKeyPairService) {
				m.On("DescribeKeyPairs", mock.Anything, mock.AnythingOfType("*entity.DescribeKeyPairsRequest")).
					Return([]entity.KeyPair{
						{
							ID:          "kp-123",
							Name:        "keypair-1",
							Algorithm:   "ed25519",
							PublicKey:   "ssh-ed25519 AAAA...",
							Fingerprint: "SHA256:xxxxx",
							CreatedAt:   "2024-01-01T00:00:00Z",
						},
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockKeyPairService)
			tc.mockSetup(mockService)

			keypairAPI := &KeyPair{
				keyPairService: mockService,
			}

			reqBody, err := json.Marshal(tc.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/keypairs/describe", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/keypairs/describe", ginx.Adapt5(keypairAPI.DescribeKeyPairs))
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNewKeyPair(t *testing.T) {
	t.Parallel()

	// 创建一个临时的 repository 用于测试
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	repo, err := repository.New(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	keypairService := service.NewKeyPairService(repo)
	keypairAPI := NewKeyPair(keypairService)

	assert.NotNil(t, keypairAPI)
	assert.NotNil(t, keypairAPI.keyPairService)
}
