# Integration Guide - Video Editing Features

This guide explains how to integrate the new video editing services into the existing `VideoService` workflow.

## Overview

The new services are implemented and ready. This guide shows how to wire them up in the main video generation pipeline.

---

## Step 1: Update VideoService Constructor

**File**: `internal/services/video.go`

Add dependencies to `VideoService`:

```go
type VideoService struct {
    fs              afero.Fs
    logger          interfaces.Logger
    transition      TransitionConfig
    encoding        *EncodingService      // NEW
    audioMixer      *AudioMixer           // NEW
    overlayService  *OverlayService       // NEW
    effectService   *EffectService        // NEW
    subtitleService *SubtitleService      // NEW
    exportService   *ExportService        // NEW
    config          *config.Config        // NEW: Full config access
}

func NewVideoService(fs afero.Fs, logger interfaces.Logger, cfg *config.Config) *VideoService {
    return &VideoService{
        fs:              fs,
        logger:          logger,
        transition:      cfg.Transition,
        encoding:        NewEncodingService(cfg.Encoding),
        audioMixer:      NewAudioMixer(fs, logger),
        overlayService:  NewOverlayService(),
        effectService:   NewEffectService(fs, logger),
        subtitleService: NewSubtitleService(fs, logger),
        exportService:   NewExportService(fs, logger),
        config:          cfg,
    }
}
```

---

## Step 2: Apply Encoding Settings

**In**: `generateSingleVideo()` method

Replace hardcoded encoding arguments with encoding service:

```go
func (s *VideoService) generateSingleVideo(slidePath, audioPath, outputPath string, targetWidth, targetHeight int) error {
    // ... existing dimension code ...
    
    // Build encoding arguments
    encodingArgs := s.encoding.BuildAllArgs()
    
    // Build video command
    cmd := exec.Command("ffmpeg", "-y", "-loop", "1", "-i", slidePath, "-i", audioPath)
    
    // Add filters
    if targetWidth != iw || targetHeight != ih {
        // ... scaling filter ...
    }
    
    // Add encoding args
    cmd.Args = append(cmd.Args, encodingArgs...)
    cmd.Args = append(cmd.Args, outputPath)
    
    // ... execute ...
}
```

---

## Step 3: Apply Visual Effects

**New method**: `applyEffects()`

```go
func (s *VideoService) applyEffects(slidePath string, slideIndex int) (string, error) {
    outputPath := slidePath
    
    for _, effect := range s.config.Effects {
        // Check if effect applies to this slide
        slides := effect.ParseSlides(/* total slides */)
        shouldApply := false
        for _, idx := range slides {
            if idx == slideIndex {
                shouldApply = true
                break
            }
        }
        
        if !shouldApply {
            continue
        }
        
        // Apply effect based on type
        switch effect.Type {
        case "ken-burns":
            tempPath := outputPath + ".kenburns.mp4"
            duration := /* get from audio */
            if err := s.effectService.ApplyKenBurns(context.Background(), outputPath, tempPath, duration, effect); err != nil {
                return "", err
            }
            outputPath = tempPath
            
        case "text-overlay":
            // Applied in filter chain during video generation
            
        case "color-grade":
            // Applied in filter chain during video generation
            
        // ... other effects ...
        }
    }
    
    return outputPath, nil
}
```

---

## Step 4: Build Filter Chain

**New method**: `buildFilterChain()`

```go
func (s *VideoService) buildFilterChain(slideIndex int, baseFilter string) string {
    filters := []string{baseFilter}
    
    for _, effect := range s.config.Effects {
        slides := effect.ParseSlides(/* total */)
        shouldApply := false
        for _, idx := range slides {
            if idx == slideIndex {
                shouldApply = true
                break
            }
        }
        
        if !shouldApply {
            continue
        }
        
        switch effect.Type {
        case "text-overlay":
            filter := s.overlayService.BuildTextOverlayFilter(effect)
            if filter != "" {
                filters = append(filters, filter)
            }
            
        case "color-grade":
            filter := s.effectService.BuildColorGradeFilter(effect)
            if filter != "" {
                filters = append(filters, filter)
            }
            
        case "vignette":
            filter := s.effectService.BuildVignetteFilter(effect)
            if filter != "" {
                filters = append(filters, filter)
            }
            
        case "film-grain":
            filter := s.effectService.BuildFilmGrainFilter(effect)
            if filter != "" {
                filters = append(filters, filter)
            }
        }
    }
    
    // Join all filters with comma
    return strings.Join(filters, ",")
}
```

---

## Step 5: Apply Background Music

**New method**: `applyBackgroundMusic()`

```go
func (s *VideoService) applyBackgroundMusic(videoPath, outputPath string) error {
    if !s.config.Audio.BackgroundMusic.Enabled {
        return nil
    }
    
    cfg := s.config.Audio.BackgroundMusic
    if cfg.File == "" {
        s.logger.Warn("Background music enabled but no file specified")
        return nil
    }
    
    // Apply background music
    return s.audioMixer.MixBackgroundMusic(
        context.Background(),
        videoPath,
        cfg.File,
        outputPath,
        cfg,
    )
}
```

---

## Step 6: Generate Subtitles

**New method**: `generateSubtitles()`

```go
func (s *VideoService) generateSubtitles(texts []string, durations []float64, lang string, videoPath string) error {
    if !s.config.Subtitles.Enabled {
        return nil
    }
    
    // Create subtitle segments
    segments := s.subtitleService.CreateSegmentsFromTexts(texts, durations)
    
    // Generate SRT file
    srtPath := strings.TrimSuffix(videoPath, ".mp4") + "_" + lang + ".srt"
    if err := s.subtitleService.GenerateSRT(segments, srtPath); err != nil {
        return err
    }
    
    // Burn in if configured
    if s.config.Subtitles.BurnIn {
        tempPath := videoPath + ".subs.mp4"
        if err := s.subtitleService.BurnSubtitles(
            context.Background(),
            videoPath,
            srtPath,
            tempPath,
            s.config.Subtitles,
        ); err != nil {
            return err
        }
        
        // Replace original with subtitled version
        if err := s.fs.Rename(tempPath, videoPath); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## Step 7: Multi-Format Export

**New method**: `exportFormats()`

```go
func (s *VideoService) exportFormats(inputPath, basePath string) error {
    if len(s.config.Output.Formats) == 0 {
        return nil
    }
    
    for _, format := range s.config.Output.Formats {
        outputPath := fmt.Sprintf("%s.%s", basePath, format.Type)
        
        if err := s.exportService.ExportToFormat(
            context.Background(),
            inputPath,
            outputPath,
            format,
        ); err != nil {
            s.logger.Error("Failed to export format", "format", format.Type, "error", err)
            // Continue with other formats
            continue
        }
        
        s.logger.Info("Exported format", "format", format.Type, "output", outputPath)
    }
    
    return nil
}
```

---

## Step 8: Update Main Video Generation Flow

**In**: `GenerateFromSlides()` method

```go
func (s *VideoService) GenerateFromSlides(ctx context.Context, slides, audioPaths []string, outputPath string) error {
    // ... existing validation ...
    
    // Generate individual video segments (with effects)
    videoFiles := make([]string, len(slides))
    errors := make([]error, len(slides))
    var wg sync.WaitGroup
    
    for i := 0; i < len(slides); i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            
            videoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.mp4", idx))
            videoFiles[idx] = videoPath
            
            // Generate base video with effects
            if err := s.generateSingleVideoWithEffects(slides[idx], audioPaths[idx], videoPath, width, height, idx); err != nil {
                errors[idx] = err
            }
        }(i)
    }
    
    wg.Wait()
    
    // Check for errors
    for _, err := range errors {
        if err != nil {
            return err
        }
    }
    
    // Concatenate videos
    if err := s.concatenateVideos(videoFiles, outputPath); err != nil {
        return err
    }
    
    // Apply background music
    if s.config.Audio.BackgroundMusic.Enabled {
        tempPath := outputPath + ".music.mp4"
        if err := s.applyBackgroundMusic(outputPath, tempPath); err != nil {
            return err
        }
        if err := s.fs.Rename(tempPath, outputPath); err != nil {
            return err
        }
    }
    
    // Generate subtitles
    if s.config.Subtitles.Enabled {
        // Get texts and durations from somewhere
        texts := /* ... */
        durations := /* ... */
        if err := s.generateSubtitles(texts, durations, "en", outputPath); err != nil {
            s.logger.Warn("Failed to generate subtitles", "error", err)
        }
    }
    
    // Export to multiple formats
    if len(s.config.Output.Formats) > 0 {
        basePath := strings.TrimSuffix(outputPath, ".mp4")
        if err := s.exportFormats(outputPath, basePath); err != nil {
            s.logger.Warn("Failed to export some formats", "error", err)
        }
    }
    
    s.logger.Info("Video created successfully", "path", outputPath)
    return nil
}
```

---

## Step 9: Update VideoCreator

**File**: `internal/services/creator.go`

Pass config to VideoService:

```go
func NewVideoCreator(..., cfg *config.Config) *VideoCreator {
    return &VideoCreator{
        // ... existing fields ...
        videoService: NewVideoService(fs, logger, cfg),
        config:       cfg,
    }
}

func (vc *VideoCreator) CreateVideos(...) error {
    // ... existing code ...
    
    // Create video service with full config
    videoService := NewVideoService(vc.fs, vc.logger, vc.config)
    
    // ... rest of video creation ...
}
```

---

## Step 10: Update CLI

**File**: `internal/cli/create.go`

Pass config to services:

```go
func runCreate(...) error {
    // ... load config ...
    
    // Create services with config
    videoService := services.NewVideoService(fs, logger, cfg)
    
    creator := services.NewVideoCreator(
        fs,
        textService,
        translationService,
        audioService,
        videoService,
        slideService,
        logger,
        cfg, // Pass config
    )
    
    // ... run creation ...
}
```

---

## Testing Integration

### 1. Test Basic Integration

```bash
# Create simple config
cat > test.yaml << EOF
input:
  lang: en
output:
  languages: [en]
encoding:
  video:
    quality: medium
EOF

# Run
gocreator create --config test.yaml
```

### 2. Test Background Music

```bash
cat > music-test.yaml << EOF
input:
  lang: en
output:
  languages: [en]
audio:
  background_music:
    enabled: true
    file: test-music.mp3
    volume: 0.15
EOF

gocreator create --config music-test.yaml
```

### 3. Test Effects

```bash
cat > effects-test.yaml << EOF
input:
  lang: en
output:
  languages: [en]
effects:
  - type: text-overlay
    slides: all
    text: "Test Watermark"
    position: bottom-right
  - type: color-grade
    slides: all
    contrast: 1.1
EOF

gocreator create --config effects-test.yaml
```

### 4. Test Subtitles

```bash
cat > subtitle-test.yaml << EOF
input:
  lang: en
output:
  languages: [en]
subtitles:
  enabled: true
  burn_in: true
  style:
    font_size: 24
EOF

gocreator create --config subtitle-test.yaml
```

---

## Error Handling

Add proper error handling for each integration point:

```go
// Example: Effect application with error handling
func (s *VideoService) applyEffectsWithRecovery(slidePath string, slideIndex int) (string, error) {
    defer func() {
        if r := recover(); r != nil {
            s.logger.Error("Panic in effect application", "slide", slideIndex, "panic", r)
        }
    }()
    
    outputPath := slidePath
    
    for i, effect := range s.config.Effects {
        newPath, err := s.applySingleEffect(outputPath, effect, slideIndex)
        if err != nil {
            s.logger.Warn("Failed to apply effect, skipping",
                "effect", effect.Type,
                "slide", slideIndex,
                "error", err)
            continue // Skip failed effect, continue with others
        }
        outputPath = newPath
    }
    
    return outputPath, nil
}
```

---

## Performance Optimization

### 1. Parallel Processing

Already implemented for segment generation. Ensure effects don't break parallelism:

```go
// Effects should be applied per-segment
// Don't apply effects to final concatenated video (too slow)
```

### 2. Caching with Effects

Update cache key to include effects:

```go
func (s *VideoService) computeSegmentHash(..., effects []config.EffectConfig) (string, error) {
    h := sha256.New()
    
    // ... existing inputs ...
    
    // Add effects to hash
    effectsJSON, _ := json.Marshal(effects)
    h.Write(effectsJSON)
    
    return hex.EncodeToString(h.Sum(nil)), nil
}
```

### 3. Smart Filter Building

Build all filters in one pass:

```go
// Combine filters into single filter_complex
// Instead of: scale,overlay,eq,vignette (multiple filters)
// Do: scale,overlay,eq,vignette (single filter chain)
```

---

## Validation

Add validation before processing:

```go
func (s *VideoService) validateConfig() error {
    // Check background music file exists
    if s.config.Audio.BackgroundMusic.Enabled {
        if s.config.Audio.BackgroundMusic.File == "" {
            return fmt.Errorf("background music enabled but no file specified")
        }
        exists, _ := afero.Exists(s.fs, s.config.Audio.BackgroundMusic.File)
        if !exists {
            return fmt.Errorf("background music file not found: %s", s.config.Audio.BackgroundMusic.File)
        }
    }
    
    // Validate effects
    for _, effect := range s.config.Effects {
        if err := s.validateEffect(effect); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## Complete Example Integration

See `examples/integration-example.go` for a complete working example showing all features integrated.

---

## Next Steps

1. ✅ Services implemented
2. ⚡ **Current**: Integrate services into VideoService
3. ⚡ Create unit tests for integration
4. ⚡ Add integration tests with sample videos
5. ⚡ Update documentation with examples
6. ⚡ Performance testing
7. ⚡ Release!

---

## Support

For integration questions:
1. Check this guide
2. Review service implementations
3. See `FEATURES.md` for feature details
4. Open GitHub issue if stuck

---

**Integration Status**: Ready to integrate
**Estimated Time**: 4-8 hours
**Complexity**: Medium (mostly wiring)
**Testing Required**: Yes (comprehensive)
