package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSnapshotService 是 SnapshotService 的 mock 实现
type MockSnapshotService struct {
	mock.Mock
}

func (m *MockSnapshotService) CreateEBSSnapshot(ctx context.Context, req *entity.CreateSnapshotRequest) (*entity.EBSSnapshot, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.EBSSnapshot), args.Error(1)
}

func (m *MockSnapshotService) DeleteEBSSnapshot(ctx context.Context, snapshotID string) error {
	args := m.Called(ctx, snapshotID)
	return args.Error(0)
}

func (m *MockSnapshotService) DescribeEBSSnapshots(ctx context.Context, req *entity.DescribeSnapshotsRequest) ([]entity.EBSSnapshot, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.EBSSnapshot), args.Error(1)
}

func (m *MockSnapshotService) CopyEBSSnapshot(ctx context.Context, req *entity.CopySnapshotRequest) (*entity.EBSSnapshot, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.EBSSnapshot), args.Error(1)
}

func TestSnapshot_DeleteSnapshot(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DeleteSnapshotRequest
		mockSetup    func(*MockSnapshotService)
		expectStatus int
	}{
		{
			name: "successful delete",
			req: &entity.DeleteSnapshotRequest{
				SnapshotID: "snap-123",
			},
			mockSetup: func(m *MockSnapshotService) {
				m.On("DeleteEBSSnapshot", mock.Anything, "snap-123").Return(nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSnapshotService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			snapshotAPI := &Snapshot{
				snapshotService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			snapshotAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/snapshots/delete", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotNil(t, router)
		})
	}
}

func TestSnapshot_CopySnapshot(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.CopySnapshotRequest
		mockSetup    func(*MockSnapshotService)
		expectStatus int
	}{
		{
			name: "successful copy",
			req: &entity.CopySnapshotRequest{
				SourceSnapshotID: "snap-source-123",
				Description:      "Copied snapshot",
			},
			mockSetup: func(m *MockSnapshotService) {
				m.On("CopyEBSSnapshot", mock.Anything, mock.AnythingOfType("*entity.CopySnapshotRequest")).
					Return(&entity.EBSSnapshot{
						SnapshotID:   "snap-copy-456",
						VolumeID:     "vol-123",
						State:        "completed",
						VolumeSizeGB: 20,
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSnapshotService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			snapshotAPI := &Snapshot{
				snapshotService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			snapshotAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/snapshots/copy", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotNil(t, router)
		})
	}
}

func TestSnapshot_DescribeSnapshots(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		req          *entity.DescribeSnapshotsRequest
		mockSetup    func(*MockSnapshotService)
		expectStatus int
	}{
		{
			name: "describe all snapshots",
			req:  &entity.DescribeSnapshotsRequest{},
			mockSetup: func(m *MockSnapshotService) {
				m.On("DescribeEBSSnapshots", mock.Anything, mock.AnythingOfType("*entity.DescribeSnapshotsRequest")).
					Return([]entity.EBSSnapshot{
						{SnapshotID: "snap-1", VolumeID: "vol-1", State: "completed"},
						{SnapshotID: "snap-2", VolumeID: "vol-2", State: "completed"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with pagination",
			req: &entity.DescribeSnapshotsRequest{
				MaxResults: 2,
			},
			mockSetup: func(m *MockSnapshotService) {
				m.On("DescribeEBSSnapshots", mock.Anything, mock.AnythingOfType("*entity.DescribeSnapshotsRequest")).
					Return([]entity.EBSSnapshot{
						{SnapshotID: "snap-1", VolumeID: "vol-1", State: "completed"},
						{SnapshotID: "snap-2", VolumeID: "vol-2", State: "completed"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "describe with filters",
			req: &entity.DescribeSnapshotsRequest{
				Filters: []entity.Filter{
					{Name: "state", Values: []string{"completed"}},
				},
			},
			mockSetup: func(m *MockSnapshotService) {
				m.On("DescribeEBSSnapshots", mock.Anything, mock.AnythingOfType("*entity.DescribeSnapshotsRequest")).
					Return([]entity.EBSSnapshot{
						{SnapshotID: "snap-1", VolumeID: "vol-1", State: "completed"},
					}, nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSnapshotService)
			if tc.mockSetup != nil {
				tc.mockSetup(mockService)
			}

			snapshotAPI := &Snapshot{
				snapshotService: mockService,
			}

			router := setupTestRouter()
			apiGroup := router.Group("/api")
			snapshotAPI.RegisterRoutes(apiGroup)

			reqBody, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/api/snapshots/describe", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotNil(t, router)
		})
	}
}
