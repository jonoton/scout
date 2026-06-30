package motion

import (
	"image"
	"testing"
)

func TestMergeRectangles(t *testing.T) {
	tests := []struct {
		name              string
		rects             []image.Rectangle
		minOverlapPercent int
		mergeDistance     int
		expected          []image.Rectangle
	}{
		{
			name: "no merge when overlap is 0 and distance is 0",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(5, 5, 15, 15),
			},
			minOverlapPercent: 0,
			mergeDistance:     0,
			expected: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(5, 5, 15, 15),
			},
		},
		{
			name: "merge overlapping rectangles",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(5, 5, 15, 15),
			},
			minOverlapPercent: 10,
			mergeDistance:     0,
			expected: []image.Rectangle{
				image.Rect(0, 0, 15, 15),
			},
		},
		{
			name: "no merge when overlap is below threshold and distance is 0",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),  // Area 100
				image.Rect(9, 9, 19, 19),  // Area 100, Overlap Area 1 (1%)
			},
			minOverlapPercent: 10,
			mergeDistance:     0,
			expected: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(9, 9, 19, 19),
			},
		},
		{
			name: "chained merges",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(8, 0, 18, 10),  // overlaps with first
				image.Rect(16, 0, 26, 10), // overlaps with second
			},
			minOverlapPercent: 10,
			mergeDistance:     0,
			expected: []image.Rectangle{
				image.Rect(0, 0, 26, 10),
			},
		},
		{
			name: "merge nearby rectangles (within distance)",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(15, 0, 25, 10), // Gap of 5 pixels
			},
			minOverlapPercent: 0,
			mergeDistance:     5,
			expected: []image.Rectangle{
				image.Rect(0, 0, 25, 10),
			},
		},
		{
			name: "no merge nearby rectangles (outside distance)",
			rects: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(16, 0, 26, 10), // Gap of 6 pixels
			},
			minOverlapPercent: 0,
			mergeDistance:     5,
			expected: []image.Rectangle{
				image.Rect(0, 0, 10, 10),
				image.Rect(16, 0, 26, 10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mergeRectangles(tt.rects, tt.minOverlapPercent, tt.mergeDistance)
			if len(actual) != len(tt.expected) {
				t.Errorf("expected %d rects, got %d", len(tt.expected), len(actual))
				return
			}
			for i, r := range actual {
				if r != tt.expected[i] {
					t.Errorf("rect %d: expected %v, got %v", i, tt.expected[i], r)
				}
			}
		})
	}
}

func TestSetConfigDefaults(t *testing.T) {
	m := NewMotion("test")
	if m.noiseReduction%2 == 0 {
		t.Errorf("expected default noise reduction to be odd, got %d", m.noiseReduction)
	}
	if m.mergeOverlapPercent != 20 {
		t.Errorf("expected default mergeOverlapPercent to be 20, got %d", m.mergeOverlapPercent)
	}
	if m.mergeDistance != 10 {
		t.Errorf("expected default mergeDistance to be 10, got %d", m.mergeDistance)
	}

	detectShadows := false
	config := &Config{
		NoiseReduction:      4, // even, should become 5
		DetectShadows:       &detectShadows,
		MinMotionFrames:     3,
		GridSize:            32,
		MergeOverlapPercent: -1, // unset/default
		MergeDistance:       -1, // unset/default
		BackgroundHistory:   200,
		BackgroundThreshold: 10,
		ClosingSize:         5,
	}

	m.SetConfig(config)

	if m.noiseReduction != 5 {
		t.Errorf("expected noise reduction to be corrected to 5, got %d", m.noiseReduction)
	}
	if m.detectShadows != false {
		t.Errorf("expected detectShadows to be false, got %t", m.detectShadows)
	}
	if m.minMotionFrames != 3 {
		t.Errorf("expected minMotionFrames to be 3, got %d", m.minMotionFrames)
	}
	if m.gridSize != 32 {
		t.Errorf("expected gridSize to be 32, got %d", m.gridSize)
	}
	if m.mergeOverlapPercent != 20 {
		t.Errorf("expected mergeOverlapPercent to remain 20, got %d", m.mergeOverlapPercent)
	}
	if m.mergeDistance != 10 {
		t.Errorf("expected mergeDistance to remain 10, got %d", m.mergeDistance)
	}
	if m.backgroundHistory != 200 {
		t.Errorf("expected backgroundHistory to be 200, got %d", m.backgroundHistory)
	}
	if m.backgroundThreshold != 10 {
		t.Errorf("expected backgroundThreshold to be 10, got %d", m.backgroundThreshold)
	}
	if m.closingSize != 5 {
		t.Errorf("expected closingSize to be 5, got %d", m.closingSize)
	}

	// Now override to 15
	configOverride := &Config{
		MergeDistance: 15,
	}
	m.SetConfig(configOverride)
	if m.mergeDistance != 15 {
		t.Errorf("expected mergeDistance to be overridden to 15, got %d", m.mergeDistance)
	}

	// Now explicitly set to 0 (disabled)
	config2 := &Config{
		MergeOverlapPercent: 0,
		MergeDistance:       0,
	}
	m.SetConfig(config2)
	if m.mergeOverlapPercent != 0 {
		t.Errorf("expected mergeOverlapPercent to be overridden to 0, got %d", m.mergeOverlapPercent)
	}
	if m.mergeDistance != 0 {
		t.Errorf("expected mergeDistance to be overridden to 0, got %d", m.mergeDistance)
	}
}
