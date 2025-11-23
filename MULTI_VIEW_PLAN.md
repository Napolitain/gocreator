# Multi-View Implementation Plan

## Overview

Add support for split-screen and multi-camera layouts (Picture-in-Picture variants) to display multiple video sources simultaneously.

---

## 📋 Feature Specification

### **Use Cases**

1. **Tutorial Videos**: Screen recording + webcam presenter
2. **Interviews**: Two people side-by-side
3. **Comparisons**: Before/after, product A vs B
4. **Team Meetings**: Grid layout (2x2, 3x3)
5. **Reaction Videos**: Main content + reactor camera
6. **Gaming**: Gameplay + facecam
7. **Presentations**: Slides + presenter

---

## 🎨 Layout Types

### **1. Split Screen (Side-by-Side)**
```
┌─────────────┬─────────────┐
│             │             │
│   Video 1   │   Video 2   │
│             │             │
│             │             │
└─────────────┴─────────────┘
```

### **2. Split Screen (Top-Bottom)**
```
┌───────────────────────────┐
│         Video 1           │
│                           │
├───────────────────────────┤
│         Video 2           │
│                           │
└───────────────────────────┘
```

### **3. Picture-in-Picture (PiP)**
```
┌───────────────────────────┐
│                           │
│       Main Video          │
│                           │
│                  ┌──────┐ │
│                  │Small │ │
│                  │Video │ │
└──────────────────┴──────┴─┘
```

### **4. Grid Layout (2x2)**
```
┌─────────────┬─────────────┐
│   Video 1   │   Video 2   │
│             │             │
├─────────────┼─────────────┤
│   Video 3   │   Video 4   │
│             │             │
└─────────────┴─────────────┘
```

### **5. Focus + Gallery (Zoom-style)**
```
┌─────────────────────┬───┐
│                     │ 1 │
│    Main Speaker     ├───┤
│                     │ 2 │
│                     ├───┤
│                     │ 3 │
└─────────────────────┴───┘
```

---

## 🔧 Configuration Schema

### **YAML Configuration**

```yaml
# Basic split screen
multi_view:
  enabled: true
  
  layouts:
    # Simple 50/50 split
    - type: split-horizontal
      slides: 0-5
      ratio: 50:50
      videos:
        left: videos/presenter.mp4
        right: videos/screen-recording.mp4
      gap: 0  # pixels between videos
      
    # Vertical split
    - type: split-vertical
      slides: 6-10
      ratio: 60:40
      videos:
        top: videos/demo.mp4
        bottom: videos/webcam.mp4
      
    # Picture-in-Picture
    - type: pip
      slides: 11-15
      main: videos/screen.mp4
      overlay: videos/facecam.mp4
      position: bottom-right
      size: 20%  # of main video
      offset: [10, 10]  # pixels from edge
      border:
        width: 2
        color: white
        
    # Grid layout
    - type: grid
      slides: 16-20
      rows: 2
      cols: 2
      videos:
        - videos/person1.mp4
        - videos/person2.mp4
        - videos/person3.mp4
        - videos/person4.mp4
      gap: 4  # pixels between cells
      
    # Focus + Gallery (Zoom-style)
    - type: focus-gallery
      slides: 21-25
      focus: videos/speaker.mp4
      gallery:
        - videos/participant1.mp4
        - videos/participant2.mp4
        - videos/participant3.mp4
      gallery_position: right
      gallery_size: 20%  # of total width
```

### **Advanced Configuration**

```yaml
multi_view:
  layouts:
    # Custom positioning
    - type: custom
      slides: 0-5
      videos:
        - source: videos/main.mp4
          position: [0, 0]      # x, y
          size: [1280, 720]     # width, height
          z_index: 1
          
        - source: videos/overlay.mp4
          position: [900, 500]
          size: [320, 180]
          z_index: 2
          opacity: 0.9
          
    # Animated transitions between layouts
    - type: split-horizontal
      slides: 0-2
      transition_to_next: true
      transition_duration: 1.0
      
    # Synchronized playback control
    - type: split-horizontal
      slides: 3-5
      videos:
        left: video1.mp4
        right: video2.mp4
      sync:
        enabled: true
        offset: 0.5  # right video starts 0.5s later
        
    # Per-video effects
    - type: split-horizontal
      slides: 6-8
      videos:
        left:
          source: video1.mp4
          effects:
            - type: color-grade
              saturation: 1.2
        right:
          source: video2.mp4
          effects:
            - type: blur
              radius: 5
```

---

## 🏗️ Implementation Plan

### **Phase 1: Configuration & Data Structures** (Day 1)

#### **1.1 Create Config File**
`internal/config/multiview.go`

```go
package config

type MultiViewConfig struct {
    Enabled bool            `yaml:"enabled"`
    Layouts []LayoutConfig  `yaml:"layouts"`
}

type LayoutConfig struct {
    Type   string      `yaml:"type"`  // split-horizontal, split-vertical, pip, grid, custom
    Slides interface{} `yaml:"slides"` // "0-5" or [0, 1, 2]
    
    // Split screen
    Ratio string      `yaml:"ratio,omitempty"`  // "50:50", "60:40"
    Videos VideoSources `yaml:"videos,omitempty"`
    Gap    int         `yaml:"gap,omitempty"`
    
    // PiP
    Main     string       `yaml:"main,omitempty"`
    Overlay  string       `yaml:"overlay,omitempty"`
    Position string       `yaml:"position,omitempty"`
    Size     string       `yaml:"size,omitempty"`
    Offset   []int        `yaml:"offset,omitempty"`
    Border   BorderConfig `yaml:"border,omitempty"`
    
    // Grid
    Rows int      `yaml:"rows,omitempty"`
    Cols int      `yaml:"cols,omitempty"`
    
    // Custom
    CustomVideos []CustomVideoConfig `yaml:"custom_videos,omitempty"`
    
    // Sync
    Sync SyncConfig `yaml:"sync,omitempty"`
}

type VideoSources struct {
    Left   string `yaml:"left,omitempty"`
    Right  string `yaml:"right,omitempty"`
    Top    string `yaml:"top,omitempty"`
    Bottom string `yaml:"bottom,omitempty"`
}

type CustomVideoConfig struct {
    Source   string    `yaml:"source"`
    Position [2]int    `yaml:"position"` // x, y
    Size     [2]int    `yaml:"size"`     // width, height
    ZIndex   int       `yaml:"z_index,omitempty"`
    Opacity  float64   `yaml:"opacity,omitempty"`
    Effects  []EffectConfig `yaml:"effects,omitempty"`
}

type BorderConfig struct {
    Width int    `yaml:"width"`
    Color string `yaml:"color"`
}

type SyncConfig struct {
    Enabled bool    `yaml:"enabled"`
    Offset  float64 `yaml:"offset"` // seconds
}
```

#### **1.2 Update Main Config**
`internal/config/config.go`

```go
type Config struct {
    // ... existing fields ...
    MultiView MultiViewConfig `yaml:"multi_view,omitempty"`
}
```

---

### **Phase 2: Multi-View Service** (Day 2-3)

#### **2.1 Create Service**
`internal/services/multiview.go`

```go
package services

import (
    "context"
    "fmt"
    "gocreator/internal/config"
)

type MultiViewService struct {
    fs     afero.Fs
    logger interfaces.Logger
}

func NewMultiViewService(fs afero.Fs, logger interfaces.Logger) *MultiViewService {
    return &MultiViewService{
        fs:     fs,
        logger: logger,
    }
}

// BuildFilterComplex builds FFmpeg filter for multi-view layout
func (s *MultiViewService) BuildFilterComplex(layout config.LayoutConfig, outputWidth, outputHeight int) (string, error) {
    switch layout.Type {
    case "split-horizontal":
        return s.buildSplitHorizontal(layout, outputWidth, outputHeight)
    case "split-vertical":
        return s.buildSplitVertical(layout, outputWidth, outputHeight)
    case "pip":
        return s.buildPiP(layout, outputWidth, outputHeight)
    case "grid":
        return s.buildGrid(layout, outputWidth, outputHeight)
    case "focus-gallery":
        return s.buildFocusGallery(layout, outputWidth, outputHeight)
    case "custom":
        return s.buildCustom(layout, outputWidth, outputHeight)
    default:
        return "", fmt.Errorf("unknown layout type: %s", layout.Type)
    }
}

// buildSplitHorizontal creates side-by-side layout
func (s *MultiViewService) buildSplitHorizontal(layout config.LayoutConfig, w, h int) (string, error) {
    // Parse ratio (e.g., "50:50" or "60:40")
    leftRatio, rightRatio := s.parseRatio(layout.Ratio)
    
    leftWidth := int(float64(w) * leftRatio)
    rightWidth := w - leftWidth - layout.Gap
    
    // Build filter complex
    filter := fmt.Sprintf(
        "[0:v]scale=%d:%d[left];"+
        "[1:v]scale=%d:%d[right];"+
        "[left][right]hstack=inputs=2:shortest=1[out]",
        leftWidth, h,
        rightWidth, h,
    )
    
    return filter, nil
}

// buildSplitVertical creates top-bottom layout
func (s *MultiViewService) buildSplitVertical(layout config.LayoutConfig, w, h int) (string, error) {
    topRatio, bottomRatio := s.parseRatio(layout.Ratio)
    
    topHeight := int(float64(h) * topRatio)
    bottomHeight := h - topHeight - layout.Gap
    
    filter := fmt.Sprintf(
        "[0:v]scale=%d:%d[top];"+
        "[1:v]scale=%d:%d[bottom];"+
        "[top][bottom]vstack=inputs=2:shortest=1[out]",
        w, topHeight,
        w, bottomHeight,
    )
    
    return filter, nil
}

// buildPiP creates picture-in-picture layout
func (s *MultiViewService) buildPiP(layout config.LayoutConfig, w, h int) (string, error) {
    // Parse size (e.g., "20%" or "320x180")
    pipWidth, pipHeight := s.parseSize(layout.Size, w, h)
    
    // Calculate position
    x, y := s.calculatePosition(layout.Position, layout.Offset, w, h, pipWidth, pipHeight)
    
    filter := fmt.Sprintf(
        "[1:v]scale=%d:%d[pip];"+
        "[0:v][pip]overlay=%d:%d[out]",
        pipWidth, pipHeight,
        x, y,
    )
    
    // Add border if specified
    if layout.Border.Width > 0 {
        filter = s.addBorder(filter, layout.Border, pipWidth, pipHeight)
    }
    
    return filter, nil
}

// buildGrid creates grid layout (2x2, 3x3, etc.)
func (s *MultiViewService) buildGrid(layout config.LayoutConfig, w, h int) (string, error) {
    rows := layout.Rows
    cols := layout.Cols
    
    if rows <= 0 || cols <= 0 {
        return "", fmt.Errorf("invalid grid dimensions: %dx%d", rows, cols)
    }
    
    cellWidth := (w - (cols-1)*layout.Gap) / cols
    cellHeight := (h - (rows-1)*layout.Gap) / rows
    
    // Scale all inputs
    var filters []string
    for i := 0; i < rows*cols; i++ {
        filters = append(filters, fmt.Sprintf("[%d:v]scale=%d:%d[v%d]", i, cellWidth, cellHeight, i))
    }
    
    // Build grid using xstack
    var inputs string
    for i := 0; i < rows*cols; i++ {
        if i > 0 {
            inputs += "|"
        }
        col := i % cols
        row := i / cols
        x := col * (cellWidth + layout.Gap)
        y := row * (cellHeight + layout.Gap)
        inputs += fmt.Sprintf("%d_%d", x, y)
    }
    
    filter := fmt.Sprintf(
        "%s;%sxstack=inputs=%d:layout=%s[out]",
        strings.Join(filters, ";"),
        s.buildInputList(rows*cols),
        rows*cols,
        inputs,
    )
    
    return filter, nil
}

// buildFocusGallery creates Zoom-style layout
func (s *MultiViewService) buildFocusGallery(layout config.LayoutConfig, w, h int) (string, error) {
    gallerySize := s.parsePercentage(layout.GallerySize, 20)
    galleryWidth := int(float64(w) * gallerySize / 100.0)
    mainWidth := w - galleryWidth
    
    galleryCount := len(layout.Gallery)
    galleryItemHeight := h / galleryCount
    
    // Scale main video
    filter := fmt.Sprintf("[0:v]scale=%d:%d[main];", mainWidth, h)
    
    // Scale and stack gallery videos
    for i := 0; i < galleryCount; i++ {
        filter += fmt.Sprintf("[%d:v]scale=%d:%d[g%d];", i+1, galleryWidth, galleryItemHeight, i)
    }
    
    // Stack gallery
    galleryInputs := s.buildInputList(galleryCount)
    filter += fmt.Sprintf("%svstack=inputs=%d[gallery];", galleryInputs, galleryCount)
    
    // Combine main and gallery
    filter += "[main][gallery]hstack=inputs=2[out]"
    
    return filter, nil
}

// Helper functions
func (s *MultiViewService) parseRatio(ratio string) (float64, float64) {
    if ratio == "" {
        return 0.5, 0.5
    }
    
    parts := strings.Split(ratio, ":")
    if len(parts) != 2 {
        return 0.5, 0.5
    }
    
    left, _ := strconv.ParseFloat(parts[0], 64)
    right, _ := strconv.ParseFloat(parts[1], 64)
    total := left + right
    
    return left / total, right / total
}

func (s *MultiViewService) parseSize(size string, refW, refH int) (int, int) {
    if strings.HasSuffix(size, "%") {
        pct, _ := strconv.ParseFloat(strings.TrimSuffix(size, "%"), 64)
        w := int(float64(refW) * pct / 100.0)
        h := int(float64(refH) * pct / 100.0)
        return w, h
    }
    
    parts := strings.Split(size, "x")
    if len(parts) == 2 {
        w, _ := strconv.Atoi(parts[0])
        h, _ := strconv.Atoi(parts[1])
        return w, h
    }
    
    // Default to 20%
    return refW / 5, refH / 5
}

func (s *MultiViewService) calculatePosition(position string, offset []int, w, h, pipW, pipH int) (int, int) {
    offsetX, offsetY := 10, 10
    if len(offset) >= 2 {
        offsetX, offsetY = offset[0], offset[1]
    }
    
    switch position {
    case "top-left":
        return offsetX, offsetY
    case "top-right":
        return w - pipW - offsetX, offsetY
    case "bottom-left":
        return offsetX, h - pipH - offsetY
    case "bottom-right":
        return w - pipW - offsetX, h - pipH - offsetY
    case "center":
        return (w - pipW) / 2, (h - pipH) / 2
    default:
        return w - pipW - offsetX, h - pipH - offsetY
    }
}

func (s *MultiViewService) addBorder(filter string, border config.BorderConfig, w, h int) string {
    // Add border using drawbox filter
    color := border.Color
    if color == "" {
        color = "white"
    }
    
    borderFilter := fmt.Sprintf(
        "drawbox=x=0:y=0:w=%d:h=%d:color=%s:t=%d",
        w, h, color, border.Width,
    )
    
    // Insert border before overlay
    return strings.Replace(filter, "[pip];", "[pip],"+borderFilter+"[pip_bordered];", 1)
}

func (s *MultiViewService) buildInputList(count int) string {
    var inputs []string
    for i := 0; i < count; i++ {
        inputs = append(inputs, fmt.Sprintf("[v%d]", i))
    }
    return strings.Join(inputs, "")
}

func (s *MultiViewService) parsePercentage(str string, defaultVal float64) float64 {
    if str == "" {
        return defaultVal
    }
    
    val, err := strconv.ParseFloat(strings.TrimSuffix(str, "%"), 64)
    if err != nil {
        return defaultVal
    }
    return val
}
```

---

### **Phase 3: FFmpeg Integration** (Day 4)

#### **3.1 Generate Multi-View Video**
`internal/services/multiview.go` (continued)

```go
// GenerateMultiViewVideo generates video with multi-view layout
func (s *MultiViewService) GenerateMultiViewVideo(
    ctx context.Context,
    layout config.LayoutConfig,
    outputPath string,
    outputWidth, outputHeight int,
) error {
    // Build filter complex
    filterComplex, err := s.BuildFilterComplex(layout, outputWidth, outputHeight)
    if err != nil {
        return fmt.Errorf("failed to build filter: %w", err)
    }
    
    // Build FFmpeg command
    args := []string{"-y"}
    
    // Add input files
    inputs := s.getInputFiles(layout)
    for _, input := range inputs {
        args = append(args, "-i", input)
    }
    
    // Add filter complex
    args = append(args, "-filter_complex", filterComplex, "-map", "[out]")
    
    // Add audio (mix all sources or use first)
    audioFilter := s.buildAudioFilter(len(inputs))
    if audioFilter != "" {
        args = append(args, "-filter_complex", audioFilter, "-map", "[aout]")
    } else {
        args = append(args, "-map", "0:a")
    }
    
    // Encoding settings
    args = append(args,
        "-c:v", "libx264",
        "-preset", "medium",
        "-crf", "23",
        "-c:a", "aac",
        "-b:a", "192k",
        outputPath,
    )
    
    // Execute FFmpeg
    cmd := exec.CommandContext(ctx, "ffmpeg", args...)
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    
    s.logger.Debug("Generating multi-view video", "command", cmd.String())
    
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
    }
    
    s.logger.Info("Multi-view video generated", "output", outputPath)
    return nil
}

func (s *MultiViewService) getInputFiles(layout config.LayoutConfig) []string {
    var inputs []string
    
    switch layout.Type {
    case "split-horizontal":
        inputs = []string{layout.Videos.Left, layout.Videos.Right}
    case "split-vertical":
        inputs = []string{layout.Videos.Top, layout.Videos.Bottom}
    case "pip":
        inputs = []string{layout.Main, layout.Overlay}
    case "grid":
        // Grid videos are in a flat list
        for i := 0; i < layout.Rows*layout.Cols && i < len(layout.Videos); i++ {
            inputs = append(inputs, layout.Videos[i])
        }
    case "focus-gallery":
        inputs = append([]string{layout.Focus}, layout.Gallery...)
    case "custom":
        for _, v := range layout.CustomVideos {
            inputs = append(inputs, v.Source)
        }
    }
    
    return inputs
}

func (s *MultiViewService) buildAudioFilter(inputCount int) string {
    if inputCount <= 1 {
        return ""
    }
    
    // Mix all audio sources
    var inputs []string
    for i := 0; i < inputCount; i++ {
        inputs = append(inputs, fmt.Sprintf("[%d:a]", i))
    }
    
    return fmt.Sprintf("%samix=inputs=%d[aout]", strings.Join(inputs, ""), inputCount)
}
```

---

### **Phase 4: Integration with Video Service** (Day 5)

#### **4.1 Update Video Service**
`internal/services/video.go`

```go
type VideoService struct {
    // ... existing fields ...
    multiViewService *MultiViewService
}

func NewVideoService(fs afero.Fs, logger interfaces.Logger, cfg *config.Config) *VideoService {
    return &VideoService{
        // ... existing fields ...
        multiViewService: NewMultiViewService(fs, logger),
        config:           cfg,
    }
}

// Apply multi-view layouts to slides
func (s *VideoService) applyMultiViewLayouts(slides []string, outputPath string) error {
    if !s.config.MultiView.Enabled {
        return nil
    }
    
    for _, layout := range s.config.MultiView.Layouts {
        // Parse which slides this layout applies to
        slideIndices := s.parseSlideRange(layout.Slides, len(slides))
        
        for _, idx := range slideIndices {
            // Generate multi-view segment for this slide
            segmentPath := fmt.Sprintf("%s_multiview_%d.mp4", outputPath, idx)
            
            if err := s.multiViewService.GenerateMultiViewVideo(
                context.Background(),
                layout,
                segmentPath,
                1920, 1080,
            ); err != nil {
                return fmt.Errorf("failed to generate multi-view for slide %d: %w", idx, err)
            }
            
            // Replace original slide with multi-view version
            slides[idx] = segmentPath
        }
    }
    
    return nil
}
```

---

### **Phase 5: Testing & Examples** (Day 6)

#### **5.1 Create Test Configuration**
`examples/multiview-demo.yaml`

```yaml
input:
  lang: en
  
output:
  languages: [en]
  directory: ./data/out
  
# Multi-view configuration
multi_view:
  enabled: true
  
  layouts:
    # Slides 0-2: Split screen tutorial
    - type: split-horizontal
      slides: 0-2
      ratio: 50:50
      videos:
        left: videos/instructor.mp4
        right: videos/screen-demo.mp4
      gap: 4
      
    # Slides 3-5: Picture-in-picture
    - type: pip
      slides: 3-5
      main: videos/main-content.mp4
      overlay: videos/facecam.mp4
      position: bottom-right
      size: 20%
      offset: [10, 10]
      border:
        width: 2
        color: white
        
    # Slides 6-8: Team meeting grid
    - type: grid
      slides: 6-8
      rows: 2
      cols: 2
      videos:
        - videos/person1.mp4
        - videos/person2.mp4
        - videos/person3.mp4
        - videos/person4.mp4
      gap: 4

encoding:
  video:
    quality: high
```

#### **5.2 Create Unit Tests**
`internal/services/multiview_test.go`

```go
package services

import (
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
    
    expected := "[0:v]scale=1152:1080[left];[1:v]scale=768:1080[right];[left][right]hstack=inputs=2:shortest=1[out]"
    if filter != expected {
        t.Errorf("Expected: %s\nGot: %s", expected, filter)
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
        t.Errorf("Filter missing xstack: %s", filter)
    }
}
```

---

## 📊 Implementation Timeline

```
Week 1:
├── Day 1: Configuration structures ✓
├── Day 2: MultiView service (split, pip) ✓
├── Day 3: MultiView service (grid, gallery) ✓
├── Day 4: FFmpeg integration ✓
├── Day 5: Video service integration ✓
├── Day 6: Testing & examples ✓
└── Day 7: Documentation & polish ✓

Total: 1 week (7 days)
```

---

## 🎯 Success Criteria

- [x] Support split-screen (horizontal/vertical)
- [x] Support picture-in-picture
- [x] Support grid layouts (2x2, 3x3, etc.)
- [x] Support focus + gallery (Zoom-style)
- [x] Configurable via YAML
- [x] Proper audio mixing
- [x] Border/styling options
- [x] Custom positioning
- [x] Unit tests
- [x] Example configurations
- [x] Documentation

---

## 🚀 Future Enhancements

### **Phase 2 Features** (Optional)
1. **Animated transitions** between layouts
2. **Dynamic layout switching** based on active speaker
3. **Auto-cropping** to focus on faces
4. **Background blur** for webcams
5. **Virtual backgrounds**
6. **Real-time layout preview**

### **Phase 3 Features** (Advanced)
1. **AI-based framing** (auto-crop to subject)
2. **Gesture detection** (switch layouts on hand raise)
3. **Auto-zoom** to active speaker
4. **Beauty filters** for webcams
5. **Green screen** removal per video
6. **3D perspective** transitions

---

## 💡 Usage Examples

### **Example 1: Tutorial Video**
```yaml
multi_view:
  enabled: true
  layouts:
    - type: split-horizontal
      slides: all
      ratio: 60:40
      videos:
        left: screen-recording.mp4
        right: instructor-webcam.mp4
```

### **Example 2: Interview**
```yaml
multi_view:
  enabled: true
  layouts:
    - type: split-horizontal
      slides: all
      ratio: 50:50
      videos:
        left: interviewer.mp4
        right: interviewee.mp4
      gap: 2
```

### **Example 3: Gaming + Facecam**
```yaml
multi_view:
  enabled: true
  layouts:
    - type: pip
      slides: all
      main: gameplay.mp4
      overlay: facecam.mp4
      position: bottom-right
      size: 15%
      border:
        width: 3
        color: "#00FF00"
```

### **Example 4: Team Meeting**
```yaml
multi_view:
  enabled: true
  layouts:
    - type: grid
      slides: all
      rows: 3
      cols: 3
      videos:
        - person1.mp4
        - person2.mp4
        - person3.mp4
        - person4.mp4
        - person5.mp4
        - person6.mp4
        - person7.mp4
        - person8.mp4
        - person9.mp4
      gap: 2
```

---

## 📝 Notes

### **FFmpeg Commands Reference**

#### Split Screen (Horizontal)
```bash
ffmpeg -i left.mp4 -i right.mp4 \
  -filter_complex "[0:v]scale=960:1080[left];[1:v]scale=960:1080[right];[left][right]hstack=inputs=2[out]" \
  -map "[out]" -map 0:a output.mp4
```

#### Picture-in-Picture
```bash
ffmpeg -i main.mp4 -i overlay.mp4 \
  -filter_complex "[1:v]scale=384:216[pip];[0:v][pip]overlay=W-w-10:H-h-10[out]" \
  -map "[out]" -map 0:a output.mp4
```

#### Grid (2x2)
```bash
ffmpeg -i v1.mp4 -i v2.mp4 -i v3.mp4 -i v4.mp4 \
  -filter_complex "\
    [0:v]scale=960:540[v0];\
    [1:v]scale=960:540[v1];\
    [2:v]scale=960:540[v2];\
    [3:v]scale=960:540[v3];\
    [v0][v1][v2][v3]xstack=inputs=4:layout=0_0|w0_0|0_h0|w0_h0[out]" \
  -map "[out]" output.mp4
```

---

## ✅ Checklist

- [ ] Implement MultiViewConfig struct
- [ ] Create MultiViewService
- [ ] Implement split-horizontal
- [ ] Implement split-vertical
- [ ] Implement pip
- [ ] Implement grid
- [ ] Implement focus-gallery
- [ ] Implement custom layouts
- [ ] Add audio mixing
- [ ] Add border support
- [ ] Integrate with VideoService
- [ ] Create unit tests
- [ ] Create example configs
- [ ] Write documentation
- [ ] Test with real videos
- [ ] Performance optimization

---

**Status**: Ready to implement
**Estimated Time**: 1 week (7 days)
**Complexity**: Medium
**Dependencies**: None (uses existing infrastructure)
