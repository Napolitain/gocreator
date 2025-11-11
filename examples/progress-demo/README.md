# Progress Bar Demo

This demo showcases the new progress UI in GoCreator using bubbletea.

## Features

- **Multi-stage tracking**: Loading, Translation, Audio Generation, Video Assembly
- **Per-language progress**: Shows progress for each output language (en, fr, es)
- **Status icons**: ✓ (complete), → (in progress), ⋯ (pending), ✗ (failed)
- **Progress bars**: Visual progress bars with percentage
- **Elapsed time**: Tracks total time elapsed

## Running the Demo

```bash
cd examples/progress-demo
go run demo.go
```

The demo simulates a video creation process with:
- 3 languages (English, French, Spanish)
- 5 slides per language
- Realistic timing delays

## UI Elements

The progress UI displays:

1. **Title**: "GoCreator - Video Generation" with elapsed time
2. **Stages**: Each major stage of the process
   - Loading
   - Translation
   - Audio Generation
   - Video Assembly
3. **Items**: Per-language progress within each stage
4. **Overall Progress**: Total completion percentage

## Example Output

```
GoCreator - Video Generation (Elapsed: 8s)

  ✓ Loading - Loaded 5 slides

  ✓ Translation - All translations complete
    ✓ en: Using original
    ✓ fr: Translated 5 texts
    ✓ es: Translated 5 texts

  → Audio Generation [66%] - 2/3 languages
    ✓ en: Generated 5 files
    ✓ fr: Generated 5 files
    → es (50%): Generating audio...

  ⋯ Video Assembly

Overall Progress [████████████░░░░░░░░] 66%

Press q to quit (will not stop video generation)
```

## Integration

The same progress system is used in the main `gocreator create` command:

```bash
# With progress UI (default)
gocreator create

# Without progress UI
gocreator create --no-progress
```

The progress UI automatically tracks:
- Slide loading (local or Google Slides)
- Text translation for each language
- Audio generation for each language
- Video assembly for each language
