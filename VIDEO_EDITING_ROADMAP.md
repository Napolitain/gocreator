# Video Editing Features - Implementation Roadmap

This document provides a comprehensive plan for implementing advanced FFmpeg video editing features in GoCreator, all configurable via YAML.

## Table of Contents

1. [Complete YAML Configuration Schema](#complete-yaml-configuration-schema)
2. [Implementation Plan by Phase](#implementation-plan-by-phase)
3. [Technical Architecture](#technical-architecture)
4. [Implementation Details per Feature](#implementation-details-per-feature)
5. [Testing Strategy](#testing-strategy)
6. [Migration & Compatibility](#migration--compatibility)

---

## Complete YAML Configuration Schema

This is the target YAML schema supporting all video editing features:

```yaml
# gocreator.yaml - Complete Video Editing Configuration

input:
  lang: en
  source: local  # local, google-slides
  presentation_id: ""  # For google-slides source
  
output:
  languages:
    - en
    - fr
    - es
  directory: ./data/out
  
  # Multi-format export
  formats:
    - type: mp4
      resolution: 1920x1080
      quality: high
    - type: webm
      resolution: 1920x1080
      codec: vp9
    - type: gif
      resolution: 640x480
      fps: 15
      optimize: true

voice:
  model: tts-1-hd
  voice: alloy
  speed: 1.0
  
  # Per-language voice settings
  per_language:
    en:
      voice: alloy
      speed: 1.0
    fr:
      voice: nova
      speed: 0.95
    es:
      voice: shimmer
      speed: 1.05

cache:
  enabled: true
  directory: ./data/cache

# ========================================
# NEW: Video Editing Features
# ========================================

# Encoding settings
encoding:
  video:
    codec: libx264        # libx264, libx265, libvpx-vp9
    preset: medium        # ultrafast, fast, medium, slow, veryslow
    crf: 23              # 0-51 (lower = better quality)
    bitrate: auto        # auto or specific like "5M"
    fps: 30              # Frame rate
    pixel_format: yuv420p
    
  audio:
    codec: aac           # aac, mp3, opus
    bitrate: 192k        # 128k, 192k, 256k, 320k
    sample_rate: 48000   # 44100, 48000

# Transitions between slides
transition:
  type: fade             # none, fade, dissolve, wipeleft, wiperight, etc.
  duration: 0.5          # seconds
  
  # Per-slide custom transitions (overrides global)
  per_slide:
    - slides: 0-1
      type: circlecrop
      duration: 0.8
    - slides: 2-3
      type: smoothleft
      duration: 0.5

# Visual effects
effects:
  # Ken Burns effect (pan & zoom on images)
  - type: ken-burns
    slides: [0, 2, 4]    # Apply to specific slides
    zoom_start: 1.0      # Initial zoom level
    zoom_end: 1.2        # Final zoom level
    direction: random    # left, right, up, down, center, random
    
  # Text overlays
  - type: text-overlay
    slides: all          # all, or [0, 1, 2]
    text: "© 2025 Company Name"
    position: bottom-right  # top-left, top-right, bottom-left, bottom-right, center
    offset_x: 10         # Pixels from edge
    offset_y: 10
    font: Arial
    font_size: 24
    color: white
    outline_color: black
    outline_width: 2
    background_color: black
    background_opacity: 0.5
    fade_in: 0.5         # Fade in duration (seconds)
    fade_out: 0.5        # Fade out duration
    
  # Blur background (for vertical/portrait videos)
  - type: blur-background
    slides: [1, 3, 5]
    blur_radius: 20      # Blur intensity
    
  # Vignette effect
  - type: vignette
    slides: all
    intensity: 0.3       # 0.0 to 1.0
    
  # Color grading
  - type: color-grade
    slides: all
    brightness: 0.0      # -1.0 to 1.0
    contrast: 1.0        # 0.0 to 2.0
    saturation: 1.0      # 0.0 to 3.0
    hue: 0               # -180 to 180
    gamma: 1.0           # 0.1 to 10.0
    
  # Film grain
  - type: film-grain
    slides: all
    intensity: 0.3       # 0.0 to 1.0
    
  # Video stabilization (for shaky video clips)
  - type: stabilize
    slides: [2, 5]       # Only for video slides
    smoothing: 10        # 1-100

# Audio settings
audio:
  # Background music
  background_music:
    enabled: true
    file: assets/music/background.mp3
    volume: 0.15         # 0.0 to 1.0 (relative to narration)
    fade_in: 2.0         # Fade in duration (seconds)
    fade_out: 3.0        # Fade out duration
    loop: true           # Loop if shorter than video
    
  # Sound effects per slide
  sound_effects:
    - slide: 0
      file: assets/sounds/whoosh.mp3
      delay: 0.5         # Delay after slide starts (seconds)
      volume: 0.5        # 0.0 to 1.0
    - slide: 2
      file: assets/sounds/ding.mp3
      delay: 1.0
      volume: 0.7
      
  # Ducking (lower background music during narration)
  ducking:
    enabled: true
    threshold: -30       # dB threshold to trigger ducking
    ratio: 0.3           # Reduce background music to 30% during speech
    attack: 0.5          # Attack time (seconds)
    release: 1.0         # Release time (seconds)

# Subtitles
subtitles:
  enabled: true
  
  # Subtitle generation
  generate: true         # Auto-generate from narration text
  languages: all         # all, or [en, fr, es]
  
  # Burn-in (embed in video) or external files
  burn_in: true          # If false, generates .srt files only
  
  # Styling
  style:
    font: Arial
    font_size: 24
    bold: false
    italic: false
    color: white
    outline_color: black
    outline_width: 2
    shadow_color: black
    shadow_offset: 2
    background_color: black
    background_opacity: 0.5
    background_padding: 5
    position: bottom     # top, bottom, middle
    alignment: center    # left, center, right
    margin_vertical: 20  # Pixels from top/bottom
    margin_horizontal: 10 # Pixels from sides
    
  # Timing
  timing:
    max_chars_per_line: 42
    max_lines: 2
    min_duration: 1.0    # Minimum subtitle duration (seconds)
    max_duration: 7.0    # Maximum subtitle duration

# Intro/Outro
intro:
  enabled: true
  video: assets/intro.mp4  # Pre-rendered intro video
  transition: fade         # Transition to first slide
  transition_duration: 0.5
  
  # Or generate from template
  template:
    enabled: false
    type: simple         # simple, professional, animated
    text: "Welcome to our presentation"
    logo: assets/logo.png
    background_color: "#000000"
    text_color: "#FFFFFF"
    duration: 5.0

outro:
  enabled: true
  video: assets/outro.mp4
  transition: dissolve
  transition_duration: 0.7
  
  template:
    enabled: false
    type: call-to-action
    text: "Thanks for watching!"
    subtext: "Subscribe for more"
    logo: assets/logo.png
    background_color: "#1a1a1a"
    duration: 8.0

# Timing controls
timing:
  # Speed control per slide
  per_slide:
    - slide: 2
      speed: 1.5         # 1.5x faster (timelapse)
    - slide: 4
      speed: 0.5         # 0.5x slower (slow motion)
      
  # Explicit duration override
  - slide: 3
    duration: 8.0        # Force 8 seconds (trim/loop video)
    
  # Global defaults
  default_image_duration: auto  # auto (from audio) or fixed seconds
  min_slide_duration: 2.0       # Minimum duration per slide
  max_slide_duration: 30.0      # Maximum duration per slide

# Picture-in-Picture
pip:
  enabled: false
  
  overlays:
    - slides: 0-3        # Which slides to show PiP on
      video: assets/presenter.mp4
      position: bottom-right  # top-left, top-right, bottom-left, bottom-right, custom
      custom_position:
        x: null          # Pixels from left (null = auto)
        y: null          # Pixels from top
      size: 20%          # Percentage of main video width
      border:
        enabled: true
        width: 2
        color: white
      opacity: 1.0
      fade_in: 0.3
      fade_out: 0.3

# Advanced filters
filters:
  # Apply to all output or specific slides
  global:
    - type: sharpen
      intensity: 0.5     # 0.0 to 1.0
      
  per_slide:
    - slides: [1, 3]
      filters:
        - type: blur
          radius: 5      # Pixels
        - type: noise-reduction
          strength: medium  # low, medium, high

# Chapter markers (for platforms that support them)
chapters:
  enabled: true
  
  markers:
    - slide: 0
      title: "Introduction"
    - slide: 2
      title: "Key Features"
    - slide: 4
      title: "Demo"
    - slide: 6
      title: "Conclusion"

# Metadata
metadata:
  title: "My Video Title"
  description: "Video description"
  author: "Company Name"
  copyright: "© 2025 Company Name"
  tags:
    - tutorial
    - demo
    - product
  category: Education
  language: en
  
  # Thumbnail generation
  thumbnail:
    enabled: true
    source: slide      # slide, frame, custom
    slide_index: 0     # Which slide to use
    frame_time: 0.0    # Or timestamp in video (seconds)
    custom_file: ""    # Or path to custom image
    overlay_text: ""   # Optional text overlay
    
# Performance settings
performance:
  parallel_segments: true     # Generate video segments in parallel
  max_parallel_jobs: 4        # Max concurrent FFmpeg processes
  hardware_acceleration:
    enabled: false
    type: auto               # auto, nvenc, qsv, videotoolbox
  temp_directory: ./data/out/.temp
  
# Debug settings
debug:
  save_intermediate_files: false
  ffmpeg_loglevel: error      # quiet, panic, fatal, error, warning, info, verbose, debug
  show_ffmpeg_commands: false
```

---

## Implementation Plan by Phase

### **Phase 1: Foundation & Core Features** (Week 1-2)

**Goal:** Establish architecture and implement high-impact features

#### 1.1 Configuration System Enhancement
- [ ] Extend `Config` struct with new sections
- [ ] Add validation for all new config fields
- [ ] Create config migration system for backwards compatibility
- [ ] Add config examples and documentation

**Files to modify:**
- `internal/config/config.go`
- `internal/config/config_test.go`

#### 1.2 Encoding Presets
- [ ] Add `EncodingConfig` struct
- [ ] Implement quality presets (low, medium, high, ultra)
- [ ] Add codec selection (H.264, H.265, VP9)
- [ ] Add CRF and bitrate controls

**New files:**
- `internal/services/encoding.go`
- `internal/services/encoding_test.go`

**FFmpeg example:**
```go
func (s *VideoService) buildEncodingArgs(cfg EncodingConfig) []string {
    args := []string{"-c:v", cfg.Video.Codec}
    
    if cfg.Video.Preset != "" {
        args = append(args, "-preset", cfg.Video.Preset)
    }
    
    if cfg.Video.CRF > 0 {
        args = append(args, "-crf", strconv.Itoa(cfg.Video.CRF))
    }
    
    args = append(args, "-c:a", cfg.Audio.Codec, "-b:a", cfg.Audio.Bitrate)
    
    return args
}
```

#### 1.3 Background Music Support
- [ ] Add `AudioMixer` service
- [ ] Implement music overlay with volume control
- [ ] Add fade in/out support
- [ ] Add looping for short music tracks
- [ ] Implement audio ducking

**New files:**
- `internal/services/audio_mixer.go`
- `internal/services/audio_mixer_test.go`

**FFmpeg example:**
```bash
# Mix background music with narration
ffmpeg -i video.mp4 -i music.mp3 \
  -filter_complex "
    [1:a]volume=0.15,afade=t=in:st=0:d=2,afade=t=out:st=117:d=3,aloop=loop=-1:size=2e+09[music];
    [0:a][music]amix=inputs=2:duration=first:dropout_transition=2[a]
  " \
  -map 0:v -map "[a]" output.mp4
```

#### 1.4 Text Overlays (Watermarks)
- [ ] Add `OverlayService` for text rendering
- [ ] Implement position presets (corners, center)
- [ ] Add custom positioning support
- [ ] Add text styling (font, size, color, outline)
- [ ] Add fade in/out animations

**New files:**
- `internal/services/overlay.go`
- `internal/services/overlay_test.go`

**FFmpeg example:**
```bash
# Text watermark
ffmpeg -i video.mp4 -vf "drawtext=text='© 2025 Company':fontsize=24:fontcolor=white:x=w-tw-10:y=h-th-10:borderw=2:bordercolor=black" output.mp4
```

**Estimated Time:** 40-50 hours
**Priority:** HIGH
**Dependencies:** None

---

### **Phase 2: Visual Effects & Transitions** (Week 3-4)

**Goal:** Enhance visual appeal with effects and advanced transitions

#### 2.1 Extended Transition Support
- [ ] Add all 40+ xfade transition types
- [ ] Implement per-slide transition configuration
- [ ] Add transition preview generation
- [ ] Optimize transition rendering

**Files to modify:**
- `internal/services/transition.go`
- `internal/services/video.go`

**FFmpeg transitions to add:**
```
circlecrop, circleopen, circleclose
vertopen, vertclose, horzopen, horzclose
diagtl, diagtr, diagbl, diagbr
hlslice, hrslice, vuslice, vdslice
radial, smoothleft, smoothright, smoothup, smoothdown
pixelize, squeezeh, squeezev
```

#### 2.2 Ken Burns Effect
- [ ] Add `EffectService` base architecture
- [ ] Implement zoompan filter for Ken Burns
- [ ] Add direction control (pan left, right, up, down)
- [ ] Add zoom level configuration
- [ ] Randomization support for dynamic presentations

**New files:**
- `internal/services/effect.go`
- `internal/services/effect_test.go`

**FFmpeg example:**
```bash
# Ken Burns: zoom and pan
ffmpeg -loop 1 -i image.jpg -vf "zoompan=z='min(zoom+0.0015,1.5)':d=125:x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':s=1920x1080:fps=30" -t 5 output.mp4
```

#### 2.3 Color Grading & Filters
- [ ] Implement color correction filters
- [ ] Add brightness/contrast/saturation controls
- [ ] Add vignette effect
- [ ] Add film grain effect
- [ ] Create filter chain builder

**FFmpeg example:**
```bash
# Color grading
ffmpeg -i input.mp4 -vf "eq=brightness=0.1:contrast=1.1:saturation=1.2,vignette=angle=PI/4" output.mp4

# Film grain
ffmpeg -i input.mp4 -vf "noise=alls=20:allf=t+u" output.mp4
```

#### 2.4 Blur Background (for Portrait Videos)
- [ ] Detect portrait/vertical video clips
- [ ] Implement blurred background with overlay
- [ ] Add blur intensity control

**FFmpeg example:**
```bash
# Blur background for vertical video
ffmpeg -i input.mp4 -filter_complex "
  [0:v]scale=1080:1920,boxblur=20[bg];
  [0:v]scale=1080:-1[fg];
  [bg][fg]overlay=(W-w)/2:(H-h)/2
" output.mp4
```

**Estimated Time:** 35-45 hours
**Priority:** HIGH
**Dependencies:** Phase 1.2 (encoding)

---

### **Phase 3: Subtitles & Accessibility** (Week 5)

**Goal:** Add comprehensive subtitle support

#### 3.1 Subtitle Generation
- [ ] Add `SubtitleService`
- [ ] Generate SRT files from narration text
- [ ] Add word-level timing (using audio analysis or approximation)
- [ ] Support multiple output formats (SRT, VTT, ASS)
- [ ] Implement line breaking and duration limits

**New files:**
- `internal/services/subtitle.go`
- `internal/services/subtitle_test.go`

**SRT format:**
```
1
00:00:00,000 --> 00:00:03,500
Welcome to our presentation about GoCreator

2
00:00:03,500 --> 00:00:07,200
Let's explore the amazing features
```

#### 3.2 Subtitle Styling & Burn-in
- [ ] Implement ASS subtitle styling
- [ ] Add burn-in support (embed in video)
- [ ] Add external subtitle file generation
- [ ] Support multi-language subtitles

**FFmpeg example:**
```bash
# Burn subtitles with custom styling
ffmpeg -i video.mp4 -vf "subtitles=subs.srt:force_style='FontName=Arial,FontSize=24,PrimaryColour=&HFFFFFF,OutlineColour=&H000000,Outline=2,BackColour=&H80000000,MarginV=20'" output.mp4
```

**Estimated Time:** 20-25 hours
**Priority:** HIGH
**Dependencies:** Phase 1 (audio processing)

---

### **Phase 4: Intro/Outro & Templates** (Week 6)

**Goal:** Professional branding with intro/outro support

#### 4.1 Intro/Outro Integration
- [ ] Add support for pre-rendered intro/outro videos
- [ ] Implement seamless concatenation with transitions
- [ ] Add template-based generation (simple intros)
- [ ] Support logo overlays

**Files to modify:**
- `internal/services/video.go`

**FFmpeg example:**
```bash
# Concatenate intro + main video + outro
ffmpeg -i intro.mp4 -i main.mp4 -i outro.mp4 \
  -filter_complex "
    [0:v][0:a][1:v][1:a][2:v][2:a]concat=n=3:v=1:a=1[outv][outa]
  " \
  -map "[outv]" -map "[outa]" output.mp4
```

#### 4.2 Template Generation
- [ ] Create `TemplateService` for generating intros
- [ ] Add simple text+logo templates
- [ ] Add animated templates (fade in, zoom)
- [ ] Support custom backgrounds

**New files:**
- `internal/services/template.go`
- `internal/services/template_test.go`

**Estimated Time:** 15-20 hours
**Priority:** MEDIUM
**Dependencies:** Phase 1.4 (text overlays)

---

### **Phase 5: Multi-Format Export** (Week 7)

**Goal:** Export to multiple formats and platforms

#### 5.1 Format Profiles
- [ ] Add `ExportService`
- [ ] Implement WebM (VP9) export
- [ ] Implement optimized GIF export
- [ ] Add platform-specific presets (YouTube, Instagram, TikTok)

**New files:**
- `internal/services/export.go`
- `internal/services/export_test.go`

**FFmpeg examples:**
```bash
# WebM (VP9)
ffmpeg -i input.mp4 -c:v libvpx-vp9 -crf 30 -b:v 0 -c:a libopus -b:a 128k output.webm

# Optimized GIF
ffmpeg -i input.mp4 -vf "fps=15,scale=640:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -loop 0 output.gif

# Instagram square (1:1)
ffmpeg -i input.mp4 -vf "scale=1080:1080:force_original_aspect_ratio=decrease,pad=1080:1080:(ow-iw)/2:(oh-ih)/2" output.mp4

# TikTok vertical (9:16)
ffmpeg -i input.mp4 -vf "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2" output.mp4
```

#### 5.2 Thumbnail Generation
- [ ] Generate video thumbnails automatically
- [ ] Support custom thumbnail selection (slide or frame)
- [ ] Add text overlay on thumbnails

**Estimated Time:** 15-20 hours
**Priority:** MEDIUM
**Dependencies:** Phase 1.2 (encoding)

---

### **Phase 6: Advanced Features** (Week 8-9)

**Goal:** Power user features

#### 6.1 Picture-in-Picture (PiP)
- [ ] Add PiP overlay support
- [ ] Implement positioning presets
- [ ] Add custom positioning
- [ ] Add border and shadow effects

**FFmpeg example:**
```bash
# PiP overlay
ffmpeg -i main.mp4 -i pip.mp4 -filter_complex \
  "[1:v]scale=iw*0.2:ih*0.2[pip];[0:v][pip]overlay=W-w-10:H-h-10" \
  output.mp4
```

#### 6.2 Video Stabilization
- [ ] Implement vidstab filter for shaky videos
- [ ] Two-pass stabilization (detect + transform)
- [ ] Configurable smoothing levels

**FFmpeg example:**
```bash
# Pass 1: Detect
ffmpeg -i input.mp4 -vf vidstabdetect=shakiness=10:accuracy=15 -f null -

# Pass 2: Transform
ffmpeg -i input.mp4 -vf vidstabtransform=smoothing=30:input="transforms.trf" output.mp4
```

#### 6.3 Timing Controls
- [ ] Add per-slide speed control
- [ ] Add duration overrides
- [ ] Implement smart timing (min/max duration)

**FFmpeg example:**
```bash
# Speed up video 1.5x
ffmpeg -i input.mp4 -filter:v "setpts=0.67*PTS" -filter:a "atempo=1.5" output.mp4
```

#### 6.4 Chapter Markers
- [ ] Generate chapter metadata
- [ ] Export for YouTube (description format)
- [ ] Embed in MP4 metadata

**Estimated Time:** 25-30 hours
**Priority:** LOW
**Dependencies:** Phase 1, 2

---

### **Phase 7: Performance & Optimization** (Week 10)

**Goal:** Optimize rendering and resource usage

#### 7.1 Hardware Acceleration
- [ ] Add NVENC support (NVIDIA)
- [ ] Add Quick Sync support (Intel)
- [ ] Add VideoToolbox support (Apple Silicon)
- [ ] Auto-detect available acceleration

**FFmpeg example:**
```bash
# NVENC (NVIDIA)
ffmpeg -hwaccel cuda -i input.mp4 -c:v h264_nvenc -preset fast output.mp4

# Quick Sync (Intel)
ffmpeg -hwaccel qsv -i input.mp4 -c:v h264_qsv -preset fast output.mp4

# VideoToolbox (Mac)
ffmpeg -hwaccel videotoolbox -i input.mp4 -c:v h264_videotoolbox output.mp4
```

#### 7.2 Smart Caching Enhancements
- [ ] Cache effect-applied segments
- [ ] Cross-resolution caching (scale from high-res cache)
- [ ] Distributed cache support

#### 7.3 Parallel Processing Optimization
- [ ] Optimize parallel segment generation
- [ ] Add resource limits (CPU, memory)
- [ ] Add progress tracking

**Estimated Time:** 20-25 hours
**Priority:** MEDIUM
**Dependencies:** All previous phases

---

## Technical Architecture

### New Service Structure

```
internal/services/
├── audio_mixer.go         # Background music, sound effects, ducking
├── audio_mixer_test.go
├── encoding.go            # Encoding presets and quality control
├── encoding_test.go
├── effect.go              # Visual effects (Ken Burns, color grade, etc.)
├── effect_test.go
├── export.go              # Multi-format export
├── export_test.go
├── filter.go              # Filter chain builder
├── filter_test.go
├── overlay.go             # Text overlays, watermarks
├── overlay_test.go
├── subtitle.go            # Subtitle generation and styling
├── subtitle_test.go
├── template.go            # Intro/outro template generation
├── template_test.go
├── video.go               # Enhanced video service (orchestrator)
└── video_pipeline.go      # FFmpeg command builder
```

### Configuration Structure

```
internal/config/
├── config.go              # Main config struct
├── encoding.go            # Encoding config
├── effects.go             # Effects config
├── audio.go               # Audio config
├── subtitles.go           # Subtitle config
├── validation.go          # Config validation
└── migration.go           # Config migration/backwards compat
```

### FFmpeg Command Builder Pattern

```go
// Video Pipeline Builder Pattern
type VideoPipelineBuilder struct {
    inputs       []string
    filters      []string
    audioFilters []string
    outputs      map[string][]string
}

func (b *VideoPipelineBuilder) AddInput(path string) *VideoPipelineBuilder
func (b *VideoPipelineBuilder) AddFilter(filter string) *VideoPipelineBuilder
func (b *VideoPipelineBuilder) AddAudioFilter(filter string) *VideoPipelineBuilder
func (b *VideoPipelineBuilder) AddOverlay(overlay OverlayConfig) *VideoPipelineBuilder
func (b *VideoPipelineBuilder) SetEncoding(enc EncodingConfig) *VideoPipelineBuilder
func (b *VideoPipelineBuilder) Build() (*exec.Cmd, error)

// Usage:
cmd, err := NewVideoPipelineBuilder().
    AddInput(slidePath).
    AddInput(audioPath).
    AddFilter("scale=1920:1080").
    AddOverlay(watermark).
    AddAudioFilter("volume=0.5").
    SetEncoding(encodingConfig).
    Build()
```

---

## Implementation Details per Feature

### 1. Background Music Mixing

**Config:**
```yaml
audio:
  background_music:
    enabled: true
    file: assets/music/bg.mp3
    volume: 0.15
    fade_in: 2.0
    fade_out: 3.0
    loop: true
```

**Implementation:**
```go
func (s *AudioMixer) MixBackgroundMusic(videoPath, musicPath, outputPath string, cfg MusicConfig) error {
    // Get video duration
    duration, err := s.getVideoDuration(videoPath)
    if err != nil {
        return err
    }
    
    // Build filter complex
    filterComplex := fmt.Sprintf(
        "[1:a]volume=%f,afade=t=in:st=0:d=%f,afade=t=out:st=%f:d=%f,aloop=loop=-1:size=2e+09[music];"+
        "[0:a][music]amix=inputs=2:duration=first:dropout_transition=2[a]",
        cfg.Volume,
        cfg.FadeIn,
        duration-cfg.FadeOut,
        cfg.FadeOut,
    )
    
    cmd := exec.Command("ffmpeg", "-y",
        "-i", videoPath,
        "-i", musicPath,
        "-filter_complex", filterComplex,
        "-map", "0:v", "-map", "[a]",
        "-c:v", "copy", // Copy video stream (no re-encode)
        "-c:a", "aac", "-b:a", "192k",
        outputPath)
    
    return cmd.Run()
}
```

### 2. Text Overlay (Watermark)

**Config:**
```yaml
effects:
  - type: text-overlay
    text: "© 2025 Company"
    position: bottom-right
    font_size: 24
    color: white
    outline_width: 2
```

**Implementation:**
```go
func (s *OverlayService) BuildTextOverlayFilter(cfg TextOverlayConfig) string {
    // Map position to coordinates
    var x, y string
    switch cfg.Position {
    case "top-left":
        x, y = "10", "10"
    case "top-right":
        x, y = "w-tw-10", "10"
    case "bottom-left":
        x, y = "10", "h-th-10"
    case "bottom-right":
        x, y = "w-tw-10", "h-th-10"
    case "center":
        x, y = "(w-tw)/2", "(h-th)/2"
    }
    
    return fmt.Sprintf(
        "drawtext=text='%s':fontfile=%s:fontsize=%d:fontcolor=%s:x=%s:y=%s:borderw=%d:bordercolor=%s",
        cfg.Text,
        cfg.Font,
        cfg.FontSize,
        cfg.Color,
        x, y,
        cfg.OutlineWidth,
        cfg.OutlineColor,
    )
}
```

### 3. Ken Burns Effect

**Config:**
```yaml
effects:
  - type: ken-burns
    slides: [0, 2, 4]
    zoom_end: 1.3
    direction: random
```

**Implementation:**
```go
func (s *EffectService) ApplyKenBurns(imagePath, outputPath string, duration float64, cfg KenBurnsConfig) error {
    // Calculate zoom parameters
    zoomStart := cfg.ZoomStart
    zoomEnd := cfg.ZoomEnd
    
    // Calculate pan direction
    var xExpr, yExpr string
    switch cfg.Direction {
    case "left":
        xExpr = "iw/2-(iw/zoom/2)-t*10"
        yExpr = "ih/2-(ih/zoom/2)"
    case "right":
        xExpr = "iw/2-(iw/zoom/2)+t*10"
        yExpr = "ih/2-(ih/zoom/2)"
    case "up":
        xExpr = "iw/2-(iw/zoom/2)"
        yExpr = "ih/2-(ih/zoom/2)-t*10"
    case "down":
        xExpr = "iw/2-(iw/zoom/2)"
        yExpr = "ih/2-(ih/zoom/2)+t*10"
    case "center":
        xExpr = "iw/2-(iw/zoom/2)"
        yExpr = "ih/2-(ih/zoom/2)"
    }
    
    filter := fmt.Sprintf(
        "zoompan=z='if(lte(zoom,%f),zoom+0.002,%f)':d=%d:x='%s':y='%s':s=1920x1080:fps=30",
        zoomEnd, zoomEnd, int(duration*30), xExpr, yExpr,
    )
    
    cmd := exec.Command("ffmpeg", "-y",
        "-loop", "1",
        "-i", imagePath,
        "-vf", filter,
        "-t", fmt.Sprintf("%.2f", duration),
        "-c:v", "libx264",
        "-pix_fmt", "yuv420p",
        outputPath)
    
    return cmd.Run()
}
```

### 4. Subtitle Generation

**Implementation:**
```go
type SubtitleService struct {
    fs     afero.Fs
    logger interfaces.Logger
}

type SubtitleSegment struct {
    Index     int
    StartTime float64
    EndTime   float64
    Text      string
}

func (s *SubtitleService) GenerateSRT(segments []SubtitleSegment, outputPath string) error {
    var content strings.Builder
    
    for _, seg := range segments {
        content.WriteString(fmt.Sprintf("%d\n", seg.Index))
        content.WriteString(fmt.Sprintf("%s --> %s\n",
            formatSRTTime(seg.StartTime),
            formatSRTTime(seg.EndTime)))
        content.WriteString(fmt.Sprintf("%s\n\n", seg.Text))
    }
    
    return afero.WriteFile(s.fs, outputPath, []byte(content.String()), 0644)
}

func formatSRTTime(seconds float64) string {
    h := int(seconds / 3600)
    m := int((seconds - float64(h*3600)) / 60)
    s := int(seconds - float64(h*3600) - float64(m*60))
    ms := int((seconds - float64(int(seconds))) * 1000)
    
    return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func (s *SubtitleService) BurnSubtitles(videoPath, subtitlePath, outputPath string, cfg SubtitleStyleConfig) error {
    style := fmt.Sprintf(
        "FontName=%s,FontSize=%d,PrimaryColour=&H%s,OutlineColour=&H%s,Outline=%d,MarginV=%d",
        cfg.Font,
        cfg.FontSize,
        colorToASS(cfg.Color),
        colorToASS(cfg.OutlineColor),
        cfg.OutlineWidth,
        cfg.MarginVertical,
    )
    
    filter := fmt.Sprintf("subtitles=%s:force_style='%s'", subtitlePath, style)
    
    cmd := exec.Command("ffmpeg", "-y",
        "-i", videoPath,
        "-vf", filter,
        "-c:a", "copy",
        outputPath)
    
    return cmd.Run()
}
```

---

## Testing Strategy

### Unit Tests
- Test each service in isolation with mocks
- Test filter generation logic
- Test configuration validation
- Test FFmpeg command building

### Integration Tests
- Test complete video generation pipeline
- Test with sample assets (images, audio, videos)
- Verify output file properties (resolution, duration, codec)

### Performance Tests
- Benchmark encoding with different presets
- Measure cache effectiveness
- Profile memory usage with large projects

### Visual Quality Tests
- Generate reference videos for visual comparison
- Automated visual regression testing (optional)

---

## Migration & Compatibility

### Backwards Compatibility
- All new config fields are optional
- Default values maintain current behavior
- Old config files work without changes

### Migration Path
```go
func MigrateConfig(oldCfg *Config) *Config {
    newCfg := *oldCfg
    
    // Set defaults for new fields
    if newCfg.Encoding == nil {
        newCfg.Encoding = DefaultEncodingConfig()
    }
    
    if newCfg.Audio == nil {
        newCfg.Audio = DefaultAudioConfig()
    }
    
    return &newCfg
}
```

### Example Migration
```yaml
# Old config (still works)
input:
  lang: en
output:
  languages: [en, fr]
  
# Automatically gets defaults:
# - encoding: medium quality H.264
# - no background music
# - no effects
# - basic transitions
```

---

## Success Metrics

### Feature Completion
- [ ] All Phase 1 features implemented and tested
- [ ] All Phase 2 features implemented and tested
- [ ] Documentation complete
- [ ] Examples provided

### Performance
- Encoding speed with hardware acceleration: 2-5x faster
- Cache hit rate: >80% for iterative workflows
- Memory usage: <2GB for typical projects

### Quality
- Video quality: Visually lossless at high preset
- Audio quality: Clear speech and music mix
- Subtitle accuracy: Perfect sync with audio

---

## Next Steps

1. **Review this plan** and prioritize phases
2. **Start with Phase 1.1** - Extend configuration system
3. **Implement incrementally** - One feature at a time
4. **Test thoroughly** - Each feature before moving on
5. **Document** - Update README and examples
6. **Get feedback** - From users after each phase

---

## Appendix: FFmpeg Filter Examples

### Complete filter_complex examples:

**Multiple effects combined:**
```bash
ffmpeg -i input.mp4 \
  -filter_complex "
    [0:v]eq=brightness=0.1:contrast=1.1,
    vignette=angle=PI/4,
    drawtext=text='© 2025':fontsize=24:x=w-tw-10:y=h-th-10[v]
  " \
  -map "[v]" output.mp4
```

**Audio mixing with ducking:**
```bash
ffmpeg -i video.mp4 -i music.mp3 \
  -filter_complex "
    [1:a]volume=0.2[music];
    [0:a]asplit=2[speech][sc];
    [sc]sidechaincompress=threshold=0.03:ratio=3:attack=200:release=1000[sc];
    [music][sc]amix[a]
  " \
  -map 0:v -map "[a]" output.mp4
```

**Picture-in-picture with border:**
```bash
ffmpeg -i main.mp4 -i pip.mp4 \
  -filter_complex "
    [1:v]scale=iw*0.25:ih*0.25,
    pad=iw+4:ih+4:2:2:white[pip];
    [0:v][pip]overlay=W-w-10:H-h-10[v]
  " \
  -map "[v]" output.mp4
```

---

**Total Estimated Implementation Time:** 170-225 hours (~5-6 weeks full-time)

**Recommended Approach:** 
1. Implement Phase 1 first (foundation)
2. Get user feedback
3. Prioritize remaining phases based on demand
4. Release incrementally
