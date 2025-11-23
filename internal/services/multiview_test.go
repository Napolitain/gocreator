package services

import (
	"strings"
	"testing"

	"gocreator/internal/config"
)

func TestBuildSplitHorizontal(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type:  "split-horizontal",
		Ratio: "60:40",
		Videos: config.VideoSources{
			Left:  "left.mp4",
			Right: "right.mp4",
		},
	}

	filter, err := service.buildSplitHorizontal(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	// Check that filter contains expected components
	if !strings.Contains(filter, "scale=1152:1080") {
		t.Errorf("Filter should contain left video scale to 1152x1080, got: %s", filter)
	}

	if !strings.Contains(filter, "scale=768:1080") {
		t.Errorf("Filter should contain right video scale to 768x1080, got: %s", filter)
	}

	if !strings.Contains(filter, "hstack") {
		t.Errorf("Filter should contain hstack, got: %s", filter)
	}
}

func TestBuildSplitVertical(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type:  "split-vertical",
		Ratio: "50:50",
		Videos: config.VideoSources{
			Top:    "top.mp4",
			Bottom: "bottom.mp4",
		},
	}

	filter, err := service.buildSplitVertical(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	if !strings.Contains(filter, "vstack") {
		t.Errorf("Filter should contain vstack, got: %s", filter)
	}
}

func TestBuildPiP(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type:     "pip",
		Main:     "main.mp4",
		Overlay:  "overlay.mp4",
		Position: "bottom-right",
		Size:     "20%",
		Offset:   []int{10, 10},
	}

	filter, err := service.buildPiP(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	// Should contain scale and overlay
	if !strings.Contains(filter, "scale=") || !strings.Contains(filter, "overlay=") {
		t.Errorf("Filter missing expected components: %s", filter)
	}
}

func TestBuildPiPWithBorder(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type:     "pip",
		Main:     "main.mp4",
		Overlay:  "overlay.mp4",
		Position: "bottom-right",
		Size:     "20%",
		Border: config.BorderConfig{
			Width: 2,
			Color: "white",
		},
	}

	filter, err := service.buildPiP(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	// Should contain drawbox for border
	if !strings.Contains(filter, "drawbox") {
		t.Errorf("Filter should contain drawbox for border, got: %s", filter)
	}
}

func TestBuildGrid(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type: "grid",
		Rows: 2,
		Cols: 2,
		Gap:  4,
	}

	filter, err := service.buildGrid(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	// Should contain xstack with 4 inputs
	if !strings.Contains(filter, "xstack=inputs=4") {
		t.Errorf("Filter should contain xstack with 4 inputs, got: %s", filter)
	}
}

func TestBuildGrid3x3(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	layout := config.LayoutConfig{
		Type: "grid",
		Rows: 3,
		Cols: 3,
		Gap:  2,
	}

	filter, err := service.buildGrid(layout, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to build filter: %v", err)
	}

	// Should contain xstack with 9 inputs
	if !strings.Contains(filter, "xstack=inputs=9") {
		t.Errorf("Filter should contain xstack with 9 inputs, got: %s", filter)
	}
}

func TestParseRatio(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	tests := []struct {
		ratio     string
		wantLeft  float64
		wantRight float64
	}{
		{"50:50", 0.5, 0.5},
		{"60:40", 0.6, 0.4},
		{"70:30", 0.7, 0.3},
		{"", 0.5, 0.5}, // Default
	}

	for _, tt := range tests {
		left, right := service.parseRatio(tt.ratio)
		if left != tt.wantLeft || right != tt.wantRight {
			t.Errorf("parseRatio(%q) = (%v, %v), want (%v, %v)",
				tt.ratio, left, right, tt.wantLeft, tt.wantRight)
		}
	}
}

func TestParseSize(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	tests := []struct {
		size      string
		refW      int
		refH      int
		wantW     int
		wantH     int
	}{
		{"20%", 1920, 1080, 384, 216},
		{"25%", 1920, 1080, 480, 270},
		{"320x180", 1920, 1080, 320, 180},
		{"", 1920, 1080, 384, 216}, // Default 20%
	}

	for _, tt := range tests {
		w, h := service.parseSize(tt.size, tt.refW, tt.refH)
		if w != tt.wantW || h != tt.wantH {
			t.Errorf("parseSize(%q, %d, %d) = (%d, %d), want (%d, %d)",
				tt.size, tt.refW, tt.refH, w, h, tt.wantW, tt.wantH)
		}
	}
}

func TestCalculatePosition(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	tests := []struct {
		position string
		offset   []int
		w        int
		h        int
		pipW     int
		pipH     int
		wantX    int
		wantY    int
	}{
		{"top-left", []int{10, 10}, 1920, 1080, 384, 216, 10, 10},
		{"top-right", []int{10, 10}, 1920, 1080, 384, 216, 1526, 10},
		{"bottom-left", []int{10, 10}, 1920, 1080, 384, 216, 10, 854},
		{"bottom-right", []int{10, 10}, 1920, 1080, 384, 216, 1526, 854},
		{"center", []int{10, 10}, 1920, 1080, 384, 216, 768, 432},
	}

	for _, tt := range tests {
		x, y := service.calculatePosition(tt.position, tt.offset, tt.w, tt.h, tt.pipW, tt.pipH)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("calculatePosition(%q, %v, %d, %d, %d, %d) = (%d, %d), want (%d, %d)",
				tt.position, tt.offset, tt.w, tt.h, tt.pipW, tt.pipH, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestGetInputFiles(t *testing.T) {
	service := NewMultiViewService(nil, nil)

	tests := []struct {
		name   string
		layout config.LayoutConfig
		want   []string
	}{
		{
			name: "split-horizontal",
			layout: config.LayoutConfig{
				Type: "split-horizontal",
				Videos: config.VideoSources{
					Left:  "left.mp4",
					Right: "right.mp4",
				},
			},
			want: []string{"left.mp4", "right.mp4"},
		},
		{
			name: "pip",
			layout: config.LayoutConfig{
				Type:    "pip",
				Main:    "main.mp4",
				Overlay: "overlay.mp4",
			},
			want: []string{"main.mp4", "overlay.mp4"},
		},
		{
			name: "grid",
			layout: config.LayoutConfig{
				Type: "grid",
				GridVideos: []string{
					"v1.mp4", "v2.mp4", "v3.mp4", "v4.mp4",
				},
			},
			want: []string{"v1.mp4", "v2.mp4", "v3.mp4", "v4.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.getInputFiles(tt.layout)
			if len(got) != len(tt.want) {
				t.Errorf("getInputFiles() returned %d files, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("getInputFiles()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseSlides(t *testing.T) {
	tests := []struct {
		name        string
		slides      interface{}
		totalSlides int
		want        []int
	}{
		{
			name:        "range 0-3",
			slides:      "0-3",
			totalSlides: 10,
			want:        []int{0, 1, 2, 3},
		},
		{
			name:        "single slide",
			slides:      5,
			totalSlides: 10,
			want:        []int{5},
		},
		{
			name:        "all slides",
			slides:      "all",
			totalSlides: 5,
			want:        []int{0, 1, 2, 3, 4},
		},
		{
			name:        "array of slides",
			slides:      []interface{}{1, 3, 5},
			totalSlides: 10,
			want:        []int{1, 3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := config.LayoutConfig{
				Slides: tt.slides,
			}
			got := layout.ParseSlides(tt.totalSlides)
			if len(got) != len(tt.want) {
				t.Errorf("ParseSlides() returned %d indices, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseSlides()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
