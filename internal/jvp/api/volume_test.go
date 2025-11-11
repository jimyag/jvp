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

// MockVolumeService 是 VolumeService 的 mock 实现
type MockVolumeService struct {
	mock.Mock
}

func (m *MockVolumeService) CreateEBSVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.EBSVolume, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.EBSVolume), args.Error(1)
}

func (m *MockVolumeService) DeleteEBSVolume(ctx context.Context, volumeID string) error {
	args := m.Called(ctx, volumeID)
	return args.Error(0)
}

func (m *MockVolumeService) AttachEBSVolume(ctx context.Context, req *entity.AttachVolumeRequest) (*entity.VolumeAttachment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VolumeAttachment), args.Error(1)
}

func (m *MockVolumeService) DetachEBSVolume(ctx context.Context, req *entity.DetachVolumeRequest) (*entity.VolumeAttachment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VolumeAttachment), args.Error(1)
}

func (m *MockVolumeService) DescribeEBSVolumes(ctx context.Context, req *entity.DescribeVolumesRequest) ([]entity.EBSVolume, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.EBSVolume), args.Error(1)
}

func (m *MockVolumeService) ModifyEBSVolume(ctx context.Context, req *entity.ModifyVolumeRequest) (*entity.VolumeModification, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VolumeModification), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestVolume_CreateVolume(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.CreateVolumeRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful create",
			req: &entity.CreateVolumeRequest{
				SizeGB:     20,
				VolumeType: "gp2",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("CreateEBSVolume", mock.Anything, mock.AnythingOfType("*entity.CreateVolumeRequest")).
					Return(&entity.EBSVolume{
						VolumeID:   "vol-123",
						SizeGB:     20,
						VolumeType: "gp2",
						State:      "available",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "create from snapshot",
			req: &entity.CreateVolumeRequest{
				SizeGB:     30,
				VolumeType: "gp2",
				SnapshotID: "snap-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("CreateEBSVolume", mock.Anything, mock.AnythingOfType("*entity.CreateVolumeRequest")).
					Return(&entity.EBSVolume{
						VolumeID:   "vol-456",
						SizeGB:     30,
						VolumeType: "gp2",
						SnapshotID: "snap-123",
						State:      "available",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "create with error",
			req: &entity.CreateVolumeRequest{
				SizeGB:     20,
				VolumeType: "gp2",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("CreateEBSVolume", mock.Anything, mock.AnythingOfType("*entity.CreateVolumeRequest")).
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

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/create", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestVolume_DescribeVolumes(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DescribeVolumesRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
	}{
		{
			name: "describe all volumes",
			req:  &entity.DescribeVolumesRequest{},
			mockSetup: func(m *MockVolumeService) {
				m.On("DescribeEBSVolumes", mock.Anything, mock.AnythingOfType("*entity.DescribeVolumesRequest")).
					Return([]entity.EBSVolume{
						{VolumeID: "vol-1", SizeGB: 20, State: "available"},
						{VolumeID: "vol-2", SizeGB: 30, State: "in-use"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with pagination",
			req: &entity.DescribeVolumesRequest{
				MaxResults: 2,
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DescribeEBSVolumes", mock.Anything, mock.AnythingOfType("*entity.DescribeVolumesRequest")).
					Return([]entity.EBSVolume{
						{VolumeID: "vol-1", SizeGB: 20, State: "available"},
						{VolumeID: "vol-2", SizeGB: 30, State: "in-use"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with filters",
			req: &entity.DescribeVolumesRequest{
				Filters: []entity.Filter{
					{Name: "state", Values: []string{"available"}},
				},
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DescribeEBSVolumes", mock.Anything, mock.AnythingOfType("*entity.DescribeVolumesRequest")).
					Return([]entity.EBSVolume{
						{VolumeID: "vol-1", SizeGB: 20, State: "available"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with error",
			req:  &entity.DescribeVolumesRequest{},
			mockSetup: func(m *MockVolumeService) {
				m.On("DescribeEBSVolumes", mock.Anything, mock.AnythingOfType("*entity.DescribeVolumesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/describe", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestVolume_DeleteVolume(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DeleteVolumeRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
	}{
		{
			name: "successful delete",
			req: &entity.DeleteVolumeRequest{
				VolumeID: "vol-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DeleteEBSVolume", mock.Anything, "vol-123").Return(nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "delete with error",
			req: &entity.DeleteVolumeRequest{
				VolumeID: "vol-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DeleteEBSVolume", mock.Anything, "vol-123").Return(assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/delete", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestVolume_AttachVolume(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.AttachVolumeRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
	}{
		{
			name: "successful attach",
			req: &entity.AttachVolumeRequest{
				VolumeID:   "vol-123",
				InstanceID: "i-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("AttachEBSVolume", mock.Anything, mock.AnythingOfType("*entity.AttachVolumeRequest")).
					Return(&entity.VolumeAttachment{
						VolumeID:   "vol-123",
						InstanceID: "i-123",
						State:      "attached",
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "attach with error",
			req: &entity.AttachVolumeRequest{
				VolumeID:   "vol-123",
				InstanceID: "i-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("AttachEBSVolume", mock.Anything, mock.AnythingOfType("*entity.AttachVolumeRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/attach", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestVolume_DetachVolume(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DetachVolumeRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
	}{
		{
			name: "successful detach",
			req: &entity.DetachVolumeRequest{
				VolumeID:   "vol-123",
				InstanceID: "i-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DetachEBSVolume", mock.Anything, mock.AnythingOfType("*entity.DetachVolumeRequest")).
					Return(&entity.VolumeAttachment{
						VolumeID:   "vol-123",
						InstanceID: "i-123",
						State:      "detached",
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "detach with error",
			req: &entity.DetachVolumeRequest{
				VolumeID:   "vol-123",
				InstanceID: "i-123",
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("DetachEBSVolume", mock.Anything, mock.AnythingOfType("*entity.DetachVolumeRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/detach", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestVolume_ModifyVolume(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.ModifyVolumeRequest
		mockSetup    func(*MockVolumeService)
		expectStatus int
	}{
		{
			name: "successful modify",
			req: &entity.ModifyVolumeRequest{
				VolumeID: "vol-123",
				SizeGB:   30,
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("ModifyEBSVolume", mock.Anything, mock.AnythingOfType("*entity.ModifyVolumeRequest")).
					Return(&entity.VolumeModification{
						VolumeID:          "vol-123",
						ModificationState: "modifying",
						TargetSizeGB:      30,
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "modify with error",
			req: &entity.ModifyVolumeRequest{
				VolumeID: "vol-123",
				SizeGB:   30,
			},
			mockSetup: func(m *MockVolumeService) {
				m.On("ModifyEBSVolume", mock.Anything, mock.AnythingOfType("*entity.ModifyVolumeRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockVolumeService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			volumeAPI := &Volume{
				volumeService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			volumeAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/volumes/modify", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNewVolume(t *testing.T) {
	t.Parallel()

	t.Run("create volume API with service", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockVolumeService)
		volumeAPI := NewVolume(nil)

		assert.NotNil(t, volumeAPI)
		assert.Nil(t, volumeAPI.volumeService)

		volumeAPIWithMock := &Volume{
			volumeService: mockService,
		}
		assert.NotNil(t, volumeAPIWithMock)
		assert.Equal(t, mockService, volumeAPIWithMock.volumeService)
	})
}
