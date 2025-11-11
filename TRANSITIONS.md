# Video Transitions in GoCreator

GoCreator supports smooth transitions between slides to create more professional-looking videos. Transitions are applied between consecutive slides during video assembly.

## Overview

Transitions allow you to control how one slide transitions to the next, creating visual continuity and polish in your videos. Instead of abrupt cuts between slides, you can use effects like fades, wipes, and slides.

## Configuration

Transitions are configured in your `gocreator.yaml` configuration file:

```yaml
transition:
  type: fade        # Type of transition effect
  duration: 0.5     # Duration in seconds
```

### Available Transition Types

| Type | Description |
|------|-------------|
| `none` | No transition (direct cut) - **default** |
| `fade` | Smooth fade between slides |
| `dissolve` | Similar to fade (dissolve effect) |
| `wipeleft` | Wipe from right to left |
| `wiperight` | Wipe from left to right |
| `wipeup` | Wipe from bottom to top |
| `wipedown` | Wipe from top to bottom |
| `slideleft` | Slide from right to left |
| `slideright` | Slide from left to right |
| `slideup` | Slide from bottom to top |
| `slidedown` | Slide from top to bottom |

### Duration

- **Range**: 0.0 to 5.0 seconds
- **Default**: 0.5 seconds
- **Recommended**: 0.3 to 1.0 seconds for most presentations

**Note**: The transition duration is the overlap time between two slides. A longer duration creates a slower, more gradual transition.

## Examples

### Basic Fade Transition

```yaml
transition:
  type: fade
  duration: 0.5
```

This creates a smooth 0.5-second fade between all slides.

### Quick Wipe Transition

```yaml
transition:
  type: wipeleft
  duration: 0.3
```

Fast wipe effect, good for dynamic presentations.

### Smooth Slide Transition

```yaml
transition:
  type: slideright
  duration: 1.0
```

Slower slide effect for a more deliberate pace.

### No Transitions (Default)

```yaml
transition:
  type: none
```

Or simply omit the `transition` section entirely.

## Technical Details

### How Transitions Work

1. **Video Segments**: Each slide is first rendered with its audio into individual video segments
2. **Transition Application**: During concatenation, FFmpeg's `xfade` filter creates overlaps between segments
3. **Timing**: The transition duration is subtracted from each slide's duration to calculate the overlap
4. **Consistency**: The same transition is applied between all slides for visual consistency

### Performance Considerations

- **Processing Time**: Transitions add minimal processing time compared to simple concatenation
- **File Size**: Transitions have negligible impact on final video file size
- **Quality**: Transition quality matches your video encoding settings

### Compatibility

- **Video Slides**: Transitions work with both static image slides and video clip slides
- **Mixed Content**: You can mix images and videos in the same presentation with transitions
- **All Languages**: Transitions are applied consistently across all output language versions

## Best Practices

### Choosing a Transition Type

1. **Professional/Corporate**: Use `fade` or `dissolve` for subtle, professional transitions
2. **Educational**: Use `fade` with 0.5-0.7 second duration for clear, easy-to-follow transitions
3. **Dynamic/Marketing**: Use `wipe` or `slide` effects with 0.3-0.5 seconds for energy
4. **Technical Documentation**: Consider `none` or very short `fade` (0.2s) to minimize distraction

### Duration Guidelines

- **0.2-0.3s**: Quick, snappy transitions for fast-paced content
- **0.5s**: Standard duration, works well for most presentations (default)
- **0.7-1.0s**: Slower, more deliberate transitions for contemplative content
- **1.5s+**: Very slow, use sparingly for dramatic effect

### Consistency

For the best viewing experience, use the same transition throughout your entire video. GoCreator enforces this automatically - all slides in a video use the same transition configuration.

## Troubleshooting

### Transitions Not Appearing

1. **Check Configuration**: Ensure `type` is not `none` and `duration` is greater than 0
2. **Validate Config**: The transition config is validated; invalid settings will fall back to `none`
3. **Check Logs**: Look for warnings about invalid transition configuration

### Unexpected Behavior

1. **Single Slide**: If you have only one slide, no transitions are applied (nothing to transition between)
2. **Very Short Slides**: If your slides are very short (< 2 seconds), transitions may not be noticeable
3. **Duration Too Long**: If transition duration exceeds slide duration, it's automatically adjusted

### Error Messages

- `"transition duration must be non-negative"`: Duration cannot be negative
- `"transition duration is too long"`: Duration exceeds 5.0 seconds maximum
- `"invalid transition type"`: The specified transition type is not recognized

## Advanced Usage

### Programmatic Configuration

If using GoCreator as a library, you can set transitions programmatically:

```go
import "gocreator/internal/services"

// Create transition config
transition := services.TransitionConfig{
    Type:     services.TransitionFade,
    Duration: 0.5,
}

// Validate
if err := transition.Validate(); err != nil {
    // Handle invalid config
}

// Use in video creator config
cfg := services.VideoCreatorConfig{
    // ... other config
    Transition: transition,
}
```

### Available Types as Constants

```go
services.TransitionNone        // "none"
services.TransitionFade        // "fade"
services.TransitionWipeleft    // "wipeleft"
services.TransitionWiperight   // "wiperight"
services.TransitionWipeup      // "wipeup"
services.TransitionWipedown    // "wipedown"
services.TransitionSlideleft   // "slideleft"
services.TransitionSlideright  // "slideright"
services.TransitionSlideup     // "slideup"
services.TransitionSlidedown   // "slidedown"
services.TransitionDissolve    // "dissolve"
```

## Future Enhancements

Potential future improvements to the transition system:

1. **Per-Slide Transitions**: Different transitions for different slide pairs
2. **Custom Timing**: Independent transition timing for each transition
3. **Advanced Effects**: More complex transition effects (3D, blur, etc.)
4. **Easing Functions**: Control transition acceleration/deceleration
5. **Intro/Outro Transitions**: Special transitions for the first and last slides

## Examples

See the `examples/` directory for complete working examples with different transition configurations.

## Support

For issues or questions about transitions:
- Check the [GitHub Issues](https://github.com/Napolitain/gocreator/issues)
- Review the [main README](README.md) for general setup
- See [IMPROVEMENTS_ROADMAP.md](IMPROVEMENTS_ROADMAP.md) for planned features
