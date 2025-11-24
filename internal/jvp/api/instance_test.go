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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockInstanceService 是 InstanceService 的 mock 实现
type MockInstanceService struct {
	mock.Mock
}

func (m *MockInstanceService) RunInstance(ctx context.Context, req *entity.RunInstanceRequest) (*entity.Instance, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Instance), args.Error(1)
}

func (m *MockInstanceService) DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]entity.Instance, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Instance), args.Error(1)
}

func (m *MockInstanceService) TerminateInstances(ctx context.Context, req *entity.TerminateInstancesRequest) ([]entity.InstanceStateChange, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.InstanceStateChange), args.Error(1)
}

func (m *MockInstanceService) StopInstances(ctx context.Context, req *entity.StopInstancesRequest) ([]entity.InstanceStateChange, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.InstanceStateChange), args.Error(1)
}

func (m *MockInstanceService) StartInstances(ctx context.Context, req *entity.StartInstancesRequest) ([]entity.InstanceStateChange, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.InstanceStateChange), args.Error(1)
}

func (m *MockInstanceService) RebootInstances(ctx context.Context, req *entity.RebootInstancesRequest) ([]entity.InstanceStateChange, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.InstanceStateChange), args.Error(1)
}

func (m *MockInstanceService) ModifyInstanceAttribute(ctx context.Context, req *entity.ModifyInstanceAttributeRequest) (*entity.Instance, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Instance), args.Error(1)
}

func (m *MockInstanceService) ResetPassword(ctx context.Context, req *entity.ResetPasswordRequest) (*entity.ResetPasswordResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ResetPasswordResponse), args.Error(1)
}

func (m *MockInstanceService) GetConsoleInfo(ctx context.Context, req *entity.GetConsoleRequest) (*entity.GetConsoleResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.GetConsoleResponse), args.Error(1)
}

func TestInstance_RunInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.RunInstanceRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful run",
			req: &entity.RunInstanceRequest{
				ImageID:  "ami-123",
				MemoryMB: 2048,
				VCPUs:    2,
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("RunInstance", mock.Anything, mock.AnythingOfType("*entity.RunInstanceRequest")).
					Return(&entity.Instance{
						ID:       "i-123",
						ImageID:  "ami-123",
						MemoryMB: 2048,
						VCPUs:    2,
						State:    "running",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "run with error",
			req: &entity.RunInstanceRequest{
				ImageID:  "ami-123",
				MemoryMB: 2048,
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("RunInstance", mock.Anything, mock.AnythingOfType("*entity.RunInstanceRequest")).
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

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/run", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_DescribeInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DescribeInstancesRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "describe all instances",
			req:  &entity.DescribeInstancesRequest{},
			mockSetup: func(m *MockInstanceService) {
				m.On("DescribeInstances", mock.Anything, mock.AnythingOfType("*entity.DescribeInstancesRequest")).
					Return([]entity.Instance{
						{ID: "i-1", State: "running"},
						{ID: "i-2", State: "stopped"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with instance IDs",
			req: &entity.DescribeInstancesRequest{
				InstanceIDs: []string{"i-1", "i-2"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("DescribeInstances", mock.Anything, mock.AnythingOfType("*entity.DescribeInstancesRequest")).
					Return([]entity.Instance{
						{ID: "i-1", State: "running"},
						{ID: "i-2", State: "stopped"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with error",
			req:  &entity.DescribeInstancesRequest{},
			mockSetup: func(m *MockInstanceService) {
				m.On("DescribeInstances", mock.Anything, mock.AnythingOfType("*entity.DescribeInstancesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/describe", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_TerminateInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.TerminateInstancesRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "successful terminate",
			req: &entity.TerminateInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("TerminateInstances", mock.Anything, mock.AnythingOfType("*entity.TerminateInstancesRequest")).
					Return([]entity.InstanceStateChange{
						{InstanceID: "i-123", CurrentState: "shutting-down", PreviousState: "running"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "terminate with error",
			req: &entity.TerminateInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("TerminateInstances", mock.Anything, mock.AnythingOfType("*entity.TerminateInstancesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/terminate", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_StopInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.StopInstancesRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "successful stop",
			req: &entity.StopInstancesRequest{
				InstanceIDs: []string{"i-123"},
				Force:       false,
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("StopInstances", mock.Anything, mock.AnythingOfType("*entity.StopInstancesRequest")).
					Return([]entity.InstanceStateChange{
						{InstanceID: "i-123", CurrentState: "stopping", PreviousState: "running"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "stop with error",
			req: &entity.StopInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("StopInstances", mock.Anything, mock.AnythingOfType("*entity.StopInstancesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/stop", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_StartInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.StartInstancesRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "successful start",
			req: &entity.StartInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("StartInstances", mock.Anything, mock.AnythingOfType("*entity.StartInstancesRequest")).
					Return([]entity.InstanceStateChange{
						{InstanceID: "i-123", CurrentState: "pending", PreviousState: "stopped"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "start with error",
			req: &entity.StartInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("StartInstances", mock.Anything, mock.AnythingOfType("*entity.StartInstancesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/start", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_RebootInstances(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.RebootInstancesRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "successful reboot",
			req: &entity.RebootInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("RebootInstances", mock.Anything, mock.AnythingOfType("*entity.RebootInstancesRequest")).
					Return([]entity.InstanceStateChange{
						{InstanceID: "i-123", CurrentState: "rebooting", PreviousState: "running"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "reboot with error",
			req: &entity.RebootInstancesRequest{
				InstanceIDs: []string{"i-123"},
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("RebootInstances", mock.Anything, mock.AnythingOfType("*entity.RebootInstancesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/reboot", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestInstance_ModifyInstanceAttribute(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.ModifyInstanceAttributeRequest
		mockSetup    func(*MockInstanceService)
		expectStatus int
	}{
		{
			name: "successful modify",
			req: &entity.ModifyInstanceAttributeRequest{
				InstanceID: "i-123",
				MemoryMB:   func() *uint64 { v := uint64(2048); return &v }(),
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("ModifyInstanceAttribute", mock.Anything, mock.AnythingOfType("*entity.ModifyInstanceAttributeRequest")).
					Return(&entity.Instance{
						ID:       "i-123",
						MemoryMB: 2048,
						State:    "running",
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "modify with error",
			req: &entity.ModifyInstanceAttributeRequest{
				InstanceID: "i-123",
				MemoryMB:   func() *uint64 { v := uint64(2048); return &v }(),
			},
			mockSetup: func(m *MockInstanceService) {
				m.On("ModifyInstanceAttribute", mock.Anything, mock.AnythingOfType("*entity.ModifyInstanceAttributeRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockInstanceService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			instanceAPI := &Instance{
				instanceService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			instanceAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/instances/modify-attribute", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNewInstance(t *testing.T) {
	t.Parallel()

	t.Run("create instance API with service", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockInstanceService)
		instanceAPI := NewInstance(nil)

		assert.NotNil(t, instanceAPI)
		assert.Nil(t, instanceAPI.instanceService)

		instanceAPIWithMock := &Instance{
			instanceService: mockService,
		}
		assert.NotNil(t, instanceAPIWithMock)
		assert.Equal(t, mockService, instanceAPIWithMock.instanceService)
	})
}
