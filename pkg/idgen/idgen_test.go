package idgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	gen := New()
	assert.NotNil(t, gen)
	assert.NotNil(t, gen.sf)
}

func TestGenerateImageID(t *testing.T) {
	t.Parallel()

	gen := New()

	testcases := []struct {
		name    string
		wantErr bool
		check   func(t *testing.T, id string)
	}{
		{
			name:    "generate image ID",
			wantErr: false,
			check: func(t *testing.T, id string) {
				assert.NotEmpty(t, id)
				assert.Contains(t, id, "ami-")
			},
		},
		{
			name:    "generate multiple IDs are unique",
			wantErr: false,
			check: func(t *testing.T, id string) {
				// 生成多个 ID，确保它们是唯一的
				ids := make(map[string]bool)
				for i := 0; i < 100; i++ {
					newID, err := gen.GenerateImageID()
					require.NoError(t, err)
					assert.False(t, ids[newID], "ID should be unique: %s", newID)
					ids[newID] = true
				}
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id, err := gen.GenerateImageID()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.check != nil {
					tc.check(t, id)
				}
			}
		})
	}
}

func TestGenerateVolumeID(t *testing.T) {
	t.Parallel()

	gen := New()

	id, err := gen.GenerateVolumeID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "vol-")
}

func TestGenerateInstanceID(t *testing.T) {
	t.Parallel()

	gen := New()

	id, err := gen.GenerateInstanceID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "i-")
}

func TestGenerateID_Incremental(t *testing.T) {
	t.Parallel()

	gen := New()

	// 生成多个 ID，验证它们是递增的
	var prevID uint64
	for i := 0; i < 100; i++ {
		id, err := gen.GenerateID()
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, prevID, "ID should be incremental: %d > %d", id, prevID)
		}
		prevID = id
	}
}

func TestGenerateID_Unique(t *testing.T) {
	t.Parallel()

	gen := New()

	// 生成大量 ID，确保它们是唯一的
	ids := make(map[uint64]bool)
	for i := 0; i < 1000; i++ {
		id, err := gen.GenerateID()
		require.NoError(t, err)
		assert.False(t, ids[id], "ID should be unique: %d", id)
		ids[id] = true
	}
}

func TestDefaultGenerator(t *testing.T) {
	t.Parallel()

	gen1 := DefaultGenerator()
	gen2 := DefaultGenerator()

	// 确保返回的是同一个实例
	assert.Equal(t, gen1, gen2)
	assert.NotNil(t, gen1)
	assert.NotNil(t, gen1.sf)
}

func TestPackageLevelFunctions(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		testFn  func() (string, error)
		prefix  string
		wantErr bool
	}{
		{
			name:    "GenerateImageID",
			testFn:  GenerateImageID,
			prefix:  "ami",
			wantErr: false,
		},
		{
			name:    "GenerateVolumeID",
			testFn:  GenerateVolumeID,
			prefix:  "vol",
			wantErr: false,
		},
		{
			name:    "GenerateInstanceID",
			testFn:  GenerateInstanceID,
			prefix:  "i",
			wantErr: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id, err := tc.testFn()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
				assert.Contains(t, id, tc.prefix+"-")
			}
		})
	}
}

func TestPackageLevelGenerateID(t *testing.T) {
	t.Parallel()

	// 生成多个 ID，验证它们是递增的
	var prevID uint64
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, prevID, "ID should be incremental: %d > %d", id, prevID)
		}
		prevID = id
	}
}
