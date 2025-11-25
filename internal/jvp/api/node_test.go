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
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNodeService 是 NodeService 的 mock 实现
type MockNodeService struct {
	mock.Mock
}

func (m *MockNodeService) ListNodes(ctx context.Context) ([]*entity.Node, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Node), args.Error(1)
}

func (m *MockNodeService) DescribeNode(ctx context.Context, nodeName string) (*entity.Node, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Node), args.Error(1)
}

func (m *MockNodeService) DescribeNodeSummary(ctx context.Context, nodeName string) (*entity.NodeSummary, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.NodeSummary), args.Error(1)
}

func (m *MockNodeService) DescribeNodePCI(ctx context.Context, nodeName string) ([]entity.PCIDevice, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.PCIDevice), args.Error(1)
}

func (m *MockNodeService) DescribeNodeUSB(ctx context.Context, nodeName string) ([]entity.USBDevice, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.USBDevice), args.Error(1)
}

func (m *MockNodeService) DescribeNodeNet(ctx context.Context, nodeName string) (*service.NodeNetworkInfo, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.NodeNetworkInfo), args.Error(1)
}

func (m *MockNodeService) DescribeNodeDisks(ctx context.Context, nodeName string) ([]entity.Disk, error) {
	args := m.Called(ctx, nodeName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Disk), args.Error(1)
}

func (m *MockNodeService) EnableNode(ctx context.Context, nodeName string) error {
	args := m.Called(ctx, nodeName)
	return args.Error(0)
}

func (m *MockNodeService) DisableNode(ctx context.Context, nodeName string) error {
	args := m.Called(ctx, nodeName)
	return args.Error(0)
}

func TestNodeAPI_ListNodes(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *ListNodesRequest
		mockSetup    func(*MockNodeService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "list all nodes",
			req:  &ListNodesRequest{},
			mockSetup: func(m *MockNodeService) {
				m.On("ListNodes", mock.Anything).Return([]*entity.Node{
					{Name: "local", UUID: "local-uuid", Type: entity.NodeTypeLocal, State: entity.NodeStateOnline},
					{Name: "remote1", UUID: "remote1-uuid", Type: entity.NodeTypeRemote, State: entity.NodeStateOnline},
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "list nodes with error",
			req:  &ListNodesRequest{},
			mockSetup: func(m *MockNodeService) {
				m.On("ListNodes", mock.Anything).Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockNodeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			nodeAPI := &NodeAPI{nodeService: mockService}

			router := gin.Default()
			apiGroup := router.Group("/api")
			nodeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/list-nodes", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)

			if !tc.expectError {
				var resp ListNodesResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotNil(t, resp.Nodes)
			}
		})
	}
}

func TestNodeAPI_DescribeNode(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *DescribeNodeRequest
		mockSetup    func(*MockNodeService)
		expectStatus int
	}{
		{
			name: "describe existing node",
			req:  &DescribeNodeRequest{Name: "local"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNode", mock.Anything, "local").Return(&entity.Node{
					Name:  "local",
					UUID:  "local-uuid",
					Type:  entity.NodeTypeLocal,
					State: entity.NodeStateOnline,
				}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe non-existing node",
			req:  &DescribeNodeRequest{Name: "non-existing"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNode", mock.Anything, "non-existing").Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockNodeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			nodeAPI := &NodeAPI{nodeService: mockService}

			router := gin.Default()
			apiGroup := router.Group("/api")
			nodeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/describe-node", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNodeAPI_DescribeNodeSummary(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *DescribeNodeSummaryRequest
		mockSetup    func(*MockNodeService)
		expectStatus int
	}{
		{
			name: "get node summary",
			req:  &DescribeNodeSummaryRequest{Name: "local"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNodeSummary", mock.Anything, "local").Return(&entity.NodeSummary{
					CPU: entity.CPUInfo{
						Cores:   8,
						Threads: 16,
						Model:   "Intel Core i7",
					},
					Memory: entity.MemoryInfo{
						Total:     32 * 1024 * 1024 * 1024,
						Available: 16 * 1024 * 1024 * 1024,
					},
				}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "get summary with error",
			req:  &DescribeNodeSummaryRequest{Name: "local"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNodeSummary", mock.Anything, "local").Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockNodeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			nodeAPI := &NodeAPI{nodeService: mockService}

			router := gin.Default()
			apiGroup := router.Group("/api")
			nodeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/describe-node-summary", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNodeAPI_DescribeNodePCI(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *DescribeNodePCIRequest
		mockSetup    func(*MockNodeService)
		expectStatus int
	}{
		{
			name: "get PCI devices",
			req:  &DescribeNodePCIRequest{Name: "local"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNodePCI", mock.Anything, "local").Return([]entity.PCIDevice{
					{Address: "0000:00:00.0", Vendor: "Intel", Device: "GPU"},
				}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "get empty PCI devices",
			req:  &DescribeNodePCIRequest{Name: "local"},
			mockSetup: func(m *MockNodeService) {
				m.On("DescribeNodePCI", mock.Anything, "local").Return([]entity.PCIDevice{}, nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockNodeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			nodeAPI := &NodeAPI{nodeService: mockService}

			router := gin.Default()
			apiGroup := router.Group("/api")
			nodeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/describe-node-pci", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNodeAPI_EnableDisableNode(t *testing.T) {
	t.Parallel()

	t.Run("enable node successfully", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockNodeService)
		mockService.On("EnableNode", mock.Anything, "local").Return(nil)

		nodeAPI := &NodeAPI{nodeService: mockService}

		router := gin.Default()
		apiGroup := router.Group("/api")
		nodeAPI.RegisterRoutes(apiGroup)

		req := &EnableNodeRequest{Name: "local"}
		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/enable-node", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("disable node successfully", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockNodeService)
		mockService.On("DisableNode", mock.Anything, "local").Return(nil)

		nodeAPI := &NodeAPI{nodeService: mockService}

		router := gin.Default()
		apiGroup := router.Group("/api")
		nodeAPI.RegisterRoutes(apiGroup)

		req := &DisableNodeRequest{Name: "local"}
		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/disable-node", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("enable node with error", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockNodeService)
		mockService.On("EnableNode", mock.Anything, "local").Return(assert.AnError)

		nodeAPI := &NodeAPI{nodeService: mockService}

		router := gin.Default()
		apiGroup := router.Group("/api")
		nodeAPI.RegisterRoutes(apiGroup)

		req := &EnableNodeRequest{Name: "local"}
		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/enable-node", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestNewNodeAPI(t *testing.T) {
	t.Parallel()

	t.Run("create node API with service", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockNodeService)
		nodeAPI := &NodeAPI{nodeService: mockService}

		assert.NotNil(t, nodeAPI)
		assert.Equal(t, mockService, nodeAPI.nodeService)
	})
}
