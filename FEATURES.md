# GoCreator - Complete Feature List

This document provides a comprehensive overview of all video editing features available in GoCreator.

## Table of Contents

- [Core Features](#core-features)
- [Encoding & Quality](#encoding--quality)
- [Visual Effects](#visual-effects)
- [Audio Features](#audio-features)
- [Transitions](#transitions)
- [Subtitles](#subtitles)
- [Export Options](#export-options)
- [Advanced Features](#advanced-features)

---

## Core Features

### Multi-Language Support
- Automatic translation using OpenAI GPT
- Text-to-speech in multiple languages
- Per-language voice customization
- Subtitle generation for all languages

### Input Sources
- **Local slides**: PNG, JPEG images
- **Local videos**: MP4, MOV, AVI, MKV, WEBM clips
- **Google Slides**: Direct API integration with speaker notes
- **Mixed content**: Combine images and video clips

### Smart Caching
- Translation cache (avoid re-translation)
- Audio cache (hash-based validation)
- Video segment cache (reuse unchanged segments)
- Final video cache (skip re-concatenation)

---

## Encoding & Quality

### Quality Presets
Four built-in quality levels:

**Low** - Fast encoding, smaller files
- Resolution: 720p
- Codec: H.264, preset veryfast
- CRF: 28, Bitrate: 1M
- Audio: AAC 128k

**Medium** (Default) - Balanced quality/speed
- Resolution: 1080p
- Codec: H.264, preset medium
- CRF: 23, Bitrate: auto
- Audio: AAC 192k

**High** - Excellent quality
- Resolution: 1080p
- Codec: H.264, preset slow
- CRF: 18, Bitrate: 5M
- Audio: AAC 256k

**Ultra** - Maximum quality
- Resolution: 4K
- Codec: H.265 (HEVC), preset slow
- CRF: 16, Bitrate: 10M
- Audio: AAC 320k, 60fps

### Custom Encoding
Fine-tune all encoding parameters:
- Video codec: H.264, H.265, VP9
- Encoding preset: ultrafast to veryslow
- CRF (quality): 0-51
- Bitrate: manual or auto
- Frame rate: 24, 30, 60 fps
- Audio codec: AAC, MP3, Opus
- Audio bitrate: 128k to 320k

### Hardware Acceleration (Coming Soon)
- NVIDIA NVENC (CUDA)
- Intel Quick Sync (QSV)
- Apple VideoToolbox

---

## Visual Effects

### Ken Burns Effect
Add motion to static images with zoom and pan:
```yaml
effects:
  - type: ken-burns
    slides: [0, 2, 4]
    zoom_start: 1.0
    zoom_end: 1.3
    direction: random  # left, right, up, down, center, random
```

**Use cases:**
- Make presentations more dynamic
- Create professional photo slideshows
- Add cinematic feel to static content

### Text Overlays & Watermarks
Add text to any slide:
```yaml
effects:
  - type: text-overlay
    slides: all
    text: "© 2025 Your Company"
    position: bottom-right
    font: Arial
    font_size: 24
    color: white
    outline_color: black
    outline_width: 2
```

**Features:**
- Position presets (corners, center)
- Custom fonts and sizes
- Color and outline control
- Background box with opacity
- Fade in/out animations

### Color Grading
Adjust colors and mood:
```yaml
effects:
  - type: color-grade
    slides: all
    brightness: 0.1    # -1.0 to 1.0
    contrast: 1.1      # 0.0 to 2.0
    saturation: 1.2    # 0.0 to 3.0
    hue: 0             # -180 to 180
    gamma: 1.0         # 0.1 to 10.0
```

### Vignette Effect
Add darkened edges for focus:
```yaml
effects:
  - type: vignette
    slides: all
    intensity: 0.3  # 0.0 to 1.0
```

### Film Grain
Add cinematic film grain texture:
```yaml
effects:
  - type: film-grain
    slides: all
    intensity: 0.3
```

### Blur Background
Perfect for portrait/vertical videos:
```yaml
effects:
  - type: blur-background
    slides: [1, 3, 5]
    blur_radius: 20
```

### Video Stabilization (Coming Soon)
Stabilize shaky video clips:
```yaml
effects:
  - type: stabilize
    slides: [2, 5]
    smoothing: 10  # 1-100
```

---

## Audio Features

### Background Music
Add background music to your videos:
```yaml
audio:
  background_music:
    enabled: true
    file: assets/music/background.mp3
    volume: 0.15       # 0.0 to 1.0
    fade_in: 2.0       # seconds
    fade_out: 3.0      # seconds
    loop: true         # Loop if shorter than video
```

**Features:**
- Automatic volume balancing
- Fade in/out for smooth start/end
- Automatic looping for short tracks
- Mix with narration audio

### Sound Effects (Coming Soon)
Add sound effects at specific times:
```yaml
audio:
  sound_effects:
    - slide: 0
      file: assets/sounds/whoosh.mp3
      delay: 0.5       # seconds after slide starts
      volume: 0.5
```

### Audio Ducking (Coming Soon)
Automatically lower background music during speech:
```yaml
audio:
  ducking:
    enabled: true
    threshold: -30     # dB
    ratio: 0.3        # Reduce music to 30%
    attack: 0.5
    release: 1.0
```

---

## Transitions

### Basic Transitions
11 built-in transition types:
- `none` - Direct cut (no transition)
- `fade` - Classic fade
- `dissolve` - Similar to fade
- `wipeleft/right/up/down` - Wipe effects
- `slideleft/right/up/down` - Slide effects

### Advanced Transitions (40+ types available)
Additional FFmpeg xfade transitions:
- Circle transitions: `circlecrop`, `circleopen`, `circleclose`
- Directional: `vertopen`, `horzopen`, `diagtl`, `diagbr`
- Creative: `radial`, `smoothleft`, `pixelize`, `squeezeh`

### Per-Slide Transitions (Coming Soon)
Custom transition for each slide:
```yaml
transition:
  type: fade  # Global default
  duration: 0.5
  
  per_slide:
    - slides: 0-1
      type: circlecrop
      duration: 0.8
    - slides: 2-3
      type: smoothleft
      duration: 0.5
```

---

## Subtitles

### Automatic Generation
Generate subtitles from narration text:
```yaml
subtitles:
  enabled: true
  generate: true
  languages: all     # all, or [en, fr, es]
  burn_in: true      # Embed in video
```

### Styling
Full control over subtitle appearance:
```yaml
subtitles:
  style:
    font: Arial
    font_size: 24
    bold: false
    italic: false
    color: white
    outline_color: black
    outline_width: 2
    background_color: black
    background_opacity: 0.5
    position: bottom
    alignment: center
    margin_vertical: 20
```

### Output Formats
- **SRT** - Standard SubRip format
- **VTT** - WebVTT for web players
- **ASS** - Advanced SubStation Alpha (with styling)
- **Burn-in** - Embedded directly in video

### Timing Control
```yaml
subtitles:
  timing:
    max_chars_per_line: 42
    max_lines: 2
    min_duration: 1.0
    max_duration: 7.0
```

---

## Export Options

### Multi-Format Export
Export to multiple formats simultaneously:
```yaml
output:
  formats:
    - type: mp4
      resolution: 1920x1080
      quality: high
    
    - type: webm
      resolution: 1920x1080
      codec: libvpx-vp9
    
    - type: gif
      resolution: 640x480
      fps: 15
      optimize: true
```

### Platform-Specific Exports (Coming Soon)
Optimized presets for social media:
- **YouTube**: 1920x1080, high quality, fast start
- **Instagram**: 1080x1080 square, 60s max
- **TikTok**: 1080x1920 vertical (9:16)
- **Twitter**: 1280x720, 2:20 max

### Thumbnail Generation
Automatic thumbnail creation:
```yaml
metadata:
  thumbnail:
    enabled: true
    source: slide      # slide, frame, custom
    slide_index: 0     # Which slide to use
```

---

## Advanced Features

### Intro/Outro Videos
Add branded intro and outro:
```yaml
intro:
  enabled: true
  video: assets/intro.mp4
  transition: fade
  transition_duration: 0.5

outro:
  enabled: true
  video: assets/outro.mp4
  transition: dissolve
  transition_duration: 0.7
```

### Timing Controls (Coming Soon)
Control playback speed per slide:
```yaml
timing:
  per_slide:
    - slide: 2
      speed: 1.5     # 1.5x faster (timelapse)
    - slide: 4
      speed: 0.5     # 0.5x slower (slow motion)
    - slide: 3
      duration: 8.0  # Force specific duration
```

### Chapter Markers
Add chapters for YouTube and video players:
```yaml
chapters:
  enabled: true
  markers:
    - slide: 0
      title: "Introduction"
    - slide: 2
      title: "Key Features"
    - slide: 4
      title: "Demo"
```

### Picture-in-Picture (Coming Soon)
Overlay video in corner (e.g., presenter):
```yaml
pip:
  enabled: true
  overlays:
    - slides: 0-3
      video: assets/presenter.mp4
      position: bottom-right
      size: 20%
      border:
        enabled: true
        width: 2
        color: white
```

### Video Metadata
Embed metadata in videos:
```yaml
metadata:
  title: "My Video Title"
  description: "Video description"
  author: "Your Name"
  copyright: "© 2025 Your Company"
  tags:
    - tutorial
    - demo
  category: Education
```

---

## Configuration

All features are configured via YAML:

**Simple configuration:**
```yaml
input:
  lang: en
output:
  languages: [en, fr, es]
```

**Advanced configuration:**
See `examples/advanced-config.yaml` for a complete example with all features.

---

## Coming Soon

Features planned for future releases:

### Phase 2 (Next Release)
- Per-slide custom transitions
- All 40+ xfade transition types
- Extended Ken Burns controls

### Phase 3
- Sound effects with timing
- Audio ducking (auto-lower music during speech)
- Advanced subtitle timing

### Phase 4
- Intro/outro template generation
- Logo overlays
- Animated text

### Phase 5
- Platform-specific exports (Instagram, TikTok, etc.)
- Batch processing
- Progress tracking UI

### Phase 6
- Picture-in-picture support
- Video stabilization
- Speed ramping

### Phase 7
- Hardware acceleration (NVENC, Quick Sync, VideoToolbox)
- Distributed cache
- Performance profiling

---

## Examples

### Basic Tutorial Video
```yaml
input:
  lang: en
output:
  languages: [en]
encoding:
  video:
    quality: high
effects:
  - type: text-overlay
    text: "© 2025 Your Company"
    position: bottom-right
audio:
  background_music:
    enabled: true
    file: music.mp3
    volume: 0.15
subtitles:
  enabled: true
```

### Professional Marketing Video
```yaml
input:
  lang: en
output:
  languages: [en, fr, es, de]
  formats:
    - type: mp4
      quality: ultra
    - type: webm
encoding:
  video:
    codec: libx265
    preset: slow
    crf: 16
effects:
  - type: ken-burns
    slides: all
    zoom_end: 1.2
  - type: color-grade
    slides: all
    contrast: 1.1
    saturation: 1.15
  - type: text-overlay
    text: "© 2025 Brand"
    position: bottom-right
audio:
  background_music:
    enabled: true
    file: corporate.mp3
transition:
  type: smoothleft
  duration: 0.8
subtitles:
  enabled: true
  burn_in: true
```

### Educational Course
```yaml
input:
  lang: en
  source: google-slides
  presentation_id: "YOUR_ID"
output:
  languages: [en, es, fr]
voice:
  model: tts-1-hd
  per_language:
    en:
      voice: alloy
      speed: 0.95
    es:
      voice: nova
      speed: 1.0
effects:
  - type: text-overlay
    text: "Course Material"
    position: top-left
audio:
  background_music:
    enabled: true
    volume: 0.1
subtitles:
  enabled: true
  style:
    font_size: 26
    position: bottom
chapters:
  enabled: true
  markers:
    - slide: 0
      title: "Lesson Introduction"
    - slide: 3
      title: "Core Concepts"
    - slide: 7
      title: "Practice Examples"
```

---

## Performance

With caching enabled:
- **First run**: ~5 min for 5 slides (translation + audio + video)
- **Subsequent runs** (no changes): <10 seconds (full cache hit)
- **Partial changes**: Only regenerates affected segments
- **Cache hit rate**: Typically >70%

Encoding speed (5-slide, 5-minute video):
- **Low quality**: ~1-2 minutes
- **Medium quality**: ~3-5 minutes
- **High quality**: ~8-12 minutes
- **Ultra quality**: ~15-25 minutes

With hardware acceleration (coming soon):
- 2-5x faster encoding
- Lower CPU usage
- Real-time encoding possible

---

## API Cost Estimation

Typical costs for 5-slide video in 3 languages:

- **Translation**: ~$0.15 (300 tokens × 3 languages)
- **Audio (TTS)**: ~$0.45 (3,000 characters × 3 languages)
- **Total**: ~$0.60 per video

With caching:
- Second generation: $0.00 (cached)
- Minor text change: ~$0.10 (partial regeneration)

---

## Tips & Best Practices

### For Best Quality
1. Use `quality: high` or `ultra` in encoding
2. Use `tts-1-hd` model for clearer audio
3. Provide high-resolution source slides (1920x1080+)
4. Use background music at low volume (0.1-0.2)

### For Fastest Processing
1. Use `quality: low` or `medium`
2. Use `preset: fast` or `veryfast`
3. Enable caching
4. Process fewer languages initially

### For Smallest File Size
1. Use `quality: low`
2. Export to WebM format
3. Lower frame rate (24-30 fps)
4. Use lower audio bitrate (128k)

### For Social Media
1. Use platform-specific presets (coming soon)
2. Enable subtitles (most watch muted)
3. Keep videos short (30-60 seconds)
4. Use square or vertical format

---

## Troubleshooting

### Video too large
- Use lower quality preset
- Reduce resolution
- Use WebM format
- Lower bitrate

### Slow encoding
- Use faster preset
- Reduce resolution
- Enable hardware acceleration (coming soon)
- Process fewer languages

### Audio out of sync
- Check slide durations
- Verify audio file integrity
- Try regenerating audio cache

### FFmpeg errors
- Ensure FFmpeg is installed
- Check FFmpeg version (4.0+)
- Verify file paths and permissions
- Check available disk space

---

For more information, see the [README](README.md) and [TRANSITIONS guide](TRANSITIONS.md).
