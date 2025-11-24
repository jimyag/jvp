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

// MockImageService 是 ImageService 的 mock 实现
type MockImageService struct {
	mock.Mock
}

func (m *MockImageService) CreateImageFromInstance(ctx context.Context, req *entity.CreateImageFromInstanceRequest) (*entity.Image, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Image), args.Error(1)
}

func (m *MockImageService) DescribeImages(ctx context.Context, req *entity.DescribeImagesRequest) ([]entity.Image, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Image), args.Error(1)
}

func (m *MockImageService) RegisterImage(ctx context.Context, req *entity.RegisterImageRequest) (*entity.Image, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Image), args.Error(1)
}

func (m *MockImageService) DeleteImage(ctx context.Context, imageID string) error {
	args := m.Called(ctx, imageID)
	return args.Error(0)
}

func TestImage_CreateImage(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.CreateImageFromInstanceRequest
		mockSetup    func(*MockImageService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful create",
			req: &entity.CreateImageFromInstanceRequest{
				InstanceID: "i-123",
				ImageName:  "my-image",
			},
			mockSetup: func(m *MockImageService) {
				m.On("CreateImageFromInstance", mock.Anything, mock.AnythingOfType("*entity.CreateImageFromInstanceRequest")).
					Return(&entity.Image{
						ID:        "ami-123",
						Name:      "my-image",
						State:     "available",
						SizeGB:    20,
						Format:    "qcow2",
						CreatedAt: "2024-01-01T00:00:00Z",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "create with error",
			req: &entity.CreateImageFromInstanceRequest{
				InstanceID: "i-123",
				ImageName:  "my-image",
			},
			mockSetup: func(m *MockImageService) {
				m.On("CreateImageFromInstance", mock.Anything, mock.AnythingOfType("*entity.CreateImageFromInstanceRequest")).
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

			mockService := new(MockImageService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			imageAPI := &Image{
				imageService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			imageAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/images/create", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestImage_DescribeImages(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DescribeImagesRequest
		mockSetup    func(*MockImageService)
		expectStatus int
	}{
		{
			name: "describe all images",
			req:  &entity.DescribeImagesRequest{},
			mockSetup: func(m *MockImageService) {
				m.On("DescribeImages", mock.Anything, mock.AnythingOfType("*entity.DescribeImagesRequest")).
					Return([]entity.Image{
						{ID: "ami-1", Name: "image-1", State: "available"},
						{ID: "ami-2", Name: "image-2", State: "available"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with image IDs",
			req: &entity.DescribeImagesRequest{
				ImageIDs: []string{"ami-1", "ami-2"},
			},
			mockSetup: func(m *MockImageService) {
				m.On("DescribeImages", mock.Anything, mock.AnythingOfType("*entity.DescribeImagesRequest")).
					Return([]entity.Image{
						{ID: "ami-1", Name: "image-1", State: "available"},
						{ID: "ami-2", Name: "image-2", State: "available"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with error",
			req:  &entity.DescribeImagesRequest{},
			mockSetup: func(m *MockImageService) {
				m.On("DescribeImages", mock.Anything, mock.AnythingOfType("*entity.DescribeImagesRequest")).
					Return(nil, assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockImageService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			imageAPI := &Image{
				imageService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			imageAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/images/describe", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestImage_RegisterImage(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.RegisterImageRequest
		mockSetup    func(*MockImageService)
		expectStatus int
		expectError  bool
	}{
		{
			name: "successful register",
			req: &entity.RegisterImageRequest{
				Name: "my-image",
				Path: "/path/to/image.qcow2",
			},
			mockSetup: func(m *MockImageService) {
				m.On("RegisterImage", mock.Anything, mock.AnythingOfType("*entity.RegisterImageRequest")).
					Return(&entity.Image{
						ID:        "ami-123",
						Name:      "my-image",
						Path:      "/path/to/image.qcow2",
						State:     "available",
						SizeGB:    20,
						Format:    "qcow2",
						CreatedAt: "2024-01-01T00:00:00Z",
					}, nil)
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "register with error",
			req: &entity.RegisterImageRequest{
				Name: "my-image",
				Path: "/path/to/image.qcow2",
			},
			mockSetup: func(m *MockImageService) {
				m.On("RegisterImage", mock.Anything, mock.AnythingOfType("*entity.RegisterImageRequest")).
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

			mockService := new(MockImageService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			imageAPI := &Image{
				imageService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			imageAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/images/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestImage_DeregisterImage(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DeregisterImageRequest
		mockSetup    func(*MockImageService)
		expectStatus int
	}{
		{
			name: "successful deregister",
			req: &entity.DeregisterImageRequest{
				ImageID: "ami-123",
			},
			mockSetup: func(m *MockImageService) {
				m.On("DeleteImage", mock.Anything, "ami-123").Return(nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "deregister with error",
			req: &entity.DeregisterImageRequest{
				ImageID: "ami-123",
			},
			mockSetup: func(m *MockImageService) {
				m.On("DeleteImage", mock.Anything, "ami-123").Return(assert.AnError)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockImageService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			imageAPI := &Image{
				imageService: mockService,
			}

			router := gin.Default()
			apiGroup := router.Group("/api")
			imageAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/images/deregister", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestNewImage(t *testing.T) {
	t.Parallel()

	t.Run("create image API with service", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockImageService)
		imageAPI := NewImage(nil)

		assert.NotNil(t, imageAPI)
		assert.Nil(t, imageAPI.imageService)

		imageAPIWithMock := &Image{
			imageService: mockService,
		}
		assert.NotNil(t, imageAPIWithMock)
		assert.Equal(t, mockService, imageAPIWithMock.imageService)
	})
}
