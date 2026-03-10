# GoCreator - Local CLI Video Generator

[![CI](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml/badge.svg)](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Napolitain/gocreator/branch/main/graph/badge.svg)](https://codecov.io/gh/Napolitain/gocreator)
[![Go Report Card](https://goreportcard.com/badge/github.com/Napolitain/gocreator)](https://goreportcard.com/report/github.com/Napolitain/gocreator)

GoCreator turns local slides plus narration text into narrated videos from the command line.

## What it does

- Reads slide assets from `data/slides`
- Infers narration from matching `.txt` and audio sidecar files in `data/slides`
- Supports PNG, JPG, JPEG, PDF, MP4, MOV, AVI, MKV, and WEBM inputs
- Expands PDFs into one page per slide before rendering
- Translates narration into multiple output languages
- Generates text-to-speech audio or uses prerecorded narration
- Renders per-slide video segments and combines them into final outputs
- Caches translations, audio, video segments, and PDF preprocessing artifacts

## Current workflow

GoCreator is now local-only and CLI-first:

- `gocreator init` creates a starter project layout
- `gocreator create` runs the full generation pipeline
- Slides are discovered only from the top level of `data/slides`
- Slide ordering uses natural filename order (`slide2` comes before `slide10`)

The current media contract is:

- One narration entry is required for every final slide after PDF expansion
- Image slides and PDF pages use narration duration
- Video slides use clip duration by default
- Video slides can instead align to narration duration with `timing.media_alignment: slide`
- Video slides mix embedded clip audio with narration when the clip has audio

## Requirements

- Go 1.24+ if building from source
- `ffmpeg` and `ffprobe` in `PATH`
- `OPENAI_API_KEY` set in the environment when TTS or translation is needed
- For PDF input: `pdfinfo`, `pdfseparate`, and `pdftocairo` in `PATH`

PDF support currently relies on those PDF utilities during preprocessing so multi-page PDFs can be split and rendered into slide assets.

## Installation

### From source

```bash
go install github.com/Napolitain/gocreator/cmd/gocreator@latest
```

### Local build

```bash
go build -o gocreator.exe ./cmd/gocreator
```

## Quick start

Initialize a project:

```bash
gocreator init
```

This creates:

- `gocreator.yaml`
- `data/slides/`
- `data/out/`
- `data/cache/`

Add your assets:

- Put images, PDFs, and/or video clips in `data/slides/`
- Put matching narration files in the same folder using the same basename

Examples:

```text
data/slides/01-cover.png
data/slides/01-cover.txt

data/slides/02-demo.mp4
data/slides/02-demo.wav

data/slides/03-summary.png
data/slides/03-summary.fr.txt
```

Create videos:

```bash
gocreator create --lang en --langs-out en,fr,es
```

## Narration sidecars

GoCreator now infers narration from files placed next to each slide:

- `basename.txt`: source-language text for TTS or translation
- `basename.<lang>.txt`: language-specific text override
- `basename.mp3` / `basename.wav` / other supported audio formats: source-language prerecorded audio
- `basename.<lang>.mp3` / `basename.<lang>.wav`: language-specific prerecorded audio

Inference rules:

- If matching audio exists for the requested language, GoCreator uses it directly
- Otherwise, if matching text exists for the requested language, GoCreator uses TTS on that text
- Otherwise, if source text exists, GoCreator translates it and uses TTS
- If both text and audio exist for the same slide/language, audio wins
- Sidecars can be interleaved in any order; media ordering is driven only by slide filenames

For PDF pages, use the expanded page basename:

- `02-handout-page-0001.txt`
- `02-handout-page-0002.fr.txt`
- `02-handout-p003.wav`

`timing.media_alignment` still controls video-slide timing:

- `video`: keep the clip duration
- `slide`: trim or loop the clip to narration duration

## PDF behavior

- PDFs are discovered alongside images and videos
- A multi-page PDF is expanded into one final slide per page
- A single-page PDF still goes through the same preprocessing path
- Invalid or encrypted PDFs fail the run
- Expanded PDF artifacts are cached under `data/cache/pdf/`

If `data/slides` contains:

```text
01-cover.png
02-handout.pdf   # 3 pages
03-demo.mp4
```

then your narration sidecars could look like:

1. `01-cover.txt`
2. `02-handout-page-0001.txt`
3. `02-handout-page-0002.txt`
4. `02-handout-page-0003.txt`
5. `03-demo.wav`

## Commands

### `gocreator init`

Creates a starter config and project layout in the current directory.

### `gocreator create`

Runs the generation pipeline.

Common flags:

- `--lang`, `-l`: input language
- `--langs-out`, `-o`: comma-separated output languages
- `--config`, `-c`: config file path
- `--no-progress`: disable the progress UI

## How `create` works

1. Load slides from `data/slides`
2. Expand and cache PDF pages when needed
3. Match per-slide text and audio sidecars from `data/slides`
4. Translate only the slide texts that do not already have a target-language sidecar
5. Generate TTS only for slides that do not already have a matching audio sidecar
6. Render one video segment per final slide
7. Concatenate segments into `data/out/output-<lang>.mp4`

## Configuration notes

The config schema is larger than the currently wired runtime.

The main `create` flow actively uses:

- `input.lang`
- `output.languages`
- `cache`
- `effects`
- `transition`
- `timing.media_alignment`
- `multi_view`

Supported effects in the core pipeline are:

- `ken-burns` for still-image slides
- `text-overlay`
- `blur-background`
- `color-grade`
- `vignette`
- `film-grain`
- `stabilize` for video slides

Effects are optional. When `effects` is absent, the normal rendering path stays on the lightweight fast path.

Other config sections remain in the schema, but they are not yet part of the core `create` pipeline.

## Examples

- `examples/minimal-sidecar-tts/` - single image plus `.txt` sidecar (requires API key)
- `examples/video-prerecorded/` - single video plus prerecorded `.wav` sidecar (no API key)
- `examples/video-align-to-audio/` - video plus longer prerecorded audio using `timing.media_alignment: slide`
- `examples/language-overrides/` - mixed per-language `.txt` and prerecorded audio sidecars
- `examples/getting-started/` - five-slide starter project with matching `.txt` sidecars (requires API key)
- `examples/demo/` - two-slide multi-language CLI example with inferred sidecars (requires API key)
- `examples/demo-multiview/` - multi-view example with per-slide `.txt` sidecars (requires API key)
- `examples/*.yaml` - config schema examples

## Development

Common commands:

```bash
go mod tidy
go fmt ./...
go vet ./...
go test ./...
go build -o gocreator.exe ./cmd/gocreator
go build -o perftest.exe ./cmd/perftest
go build -o cache-perf-test.exe ./cmd/cache-perf-test
```

## Architecture

The main runtime path is:

```text
cmd/gocreator -> internal/cli/create.go -> internal/services/creator.go
```

Core services handle:

- text loading
- translation
- audio generation
- slide discovery and PDF preprocessing
- video assembly
- transitions and multi-view layout composition

## License

GPL-3.0
