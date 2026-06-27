package sortutils

import (
	"os"
	"reflect"
	"sort"
	"testing"
	"time"
)

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name    string
	modTime time.Time
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() any           { return nil }

func TestDescendingTime(t *testing.T) {
	now := time.Now()
	files := []os.FileInfo{
		mockFileInfo{name: "file_old.jpg", modTime: now.Add(-10 * time.Minute)},
		mockFileInfo{name: "file_new_a.jpg", modTime: now},
		mockFileInfo{name: "file_new_b.jpg", modTime: now},
		mockFileInfo{name: "file_mid.jpg", modTime: now.Add(-5 * time.Minute)},
	}

	sort.Sort(DescendingTime(files))

	expectedNames := []string{
		"file_new_b.jpg", // same time as file_new_a.jpg, but lexicographically greater
		"file_new_a.jpg",
		"file_mid.jpg",
		"file_old.jpg",
	}

	for i, f := range files {
		if f.Name() != expectedNames[i] {
			t.Errorf("At index %d: expected %s, got %s", i, expectedNames[i], f.Name())
		}
	}
}

func TestDescendingTimeName(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "Different timestamps",
			input: []string{
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
				"monitor_2026_06_27_13_00_00_000000000_Highlighted.jpg",
				"monitor_2026_06_27_11_00_00_000000000_Highlighted.jpg",
			},
			expected: []string{
				"monitor_2026_06_27_13_00_00_000000000_Highlighted.jpg",
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
				"monitor_2026_06_27_11_00_00_000000000_Highlighted.jpg",
			},
		},
		{
			name: "Same timestamps, different suffixes",
			input: []string{
				"monitor_2026_06_27_12_00_00_000000000_Original.jpg",
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
			},
			expected: []string{
				"monitor_2026_06_27_12_00_00_000000000_Original.jpg", // 'O' > 'H'
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
			},
		},
		{
			name: "Missing timestamps",
			input: []string{
				"no_timestamp_b.jpg",
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
				"no_timestamp_a.jpg",
			},
			expected: []string{
				"monitor_2026_06_27_12_00_00_000000000_Highlighted.jpg",
				"no_timestamp_b.jpg", // 'b' > 'a'
				"no_timestamp_a.jpg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCopy := make([]string, len(tt.input))
			copy(inputCopy, tt.input)
			sort.Sort(DescendingTimeName(inputCopy))
			if !reflect.DeepEqual(inputCopy, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, inputCopy)
			}
		})
	}
}

func TestAscendingTime(t *testing.T) {
	now := time.Now()
	files := []os.FileInfo{
		mockFileInfo{name: "file_old.jpg", modTime: now.Add(-10 * time.Minute)},
		mockFileInfo{name: "file_new_b.jpg", modTime: now},
		mockFileInfo{name: "file_new_a.jpg", modTime: now},
		mockFileInfo{name: "file_mid.jpg", modTime: now.Add(-5 * time.Minute)},
	}

	sort.Sort(AscendingTime(files))

	expectedNames := []string{
		"file_old.jpg",
		"file_mid.jpg",
		"file_new_a.jpg", // same time as file_new_b.jpg, but lexicographically smaller
		"file_new_b.jpg",
	}

	for i, f := range files {
		if f.Name() != expectedNames[i] {
			t.Errorf("At index %d: expected %s, got %s", i, expectedNames[i], f.Name())
		}
	}
}
