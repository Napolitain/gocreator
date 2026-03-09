# GoCreator - Local CLI Video Generator

[![CI](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml/badge.svg)](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Napolitain/gocreator/branch/main/graph/badge.svg)](https://codecov.io/gh/Napolitain/gocreator)
[![Go Report Card](https://goreportcard.com/badge/github.com/Napolitain/gocreator)](https://goreportcard.com/report/github.com/Napolitain/gocreator)

GoCreator turns local slides plus narration text into narrated videos from the command line.

## What it does

- Reads narration from `data/texts.txt`
- Reads slide assets from `data/slides`
- Supports PNG, JPG, JPEG, PDF, MP4, MOV, AVI, MKV, and WEBM inputs
- Expands PDFs into one page per slide before rendering
- Translates narration into multiple output languages
- Generates text-to-speech audio
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
- Video slides use clip duration
- Video slides mix embedded clip audio with generated narration

## Requirements

- Go 1.24+ if building from source
- `ffmpeg` and `ffprobe` in `PATH`
- `OPENAI_API_KEY` set in the environment
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
- `data/texts.txt`
- `data/out/`
- `data/cache/`

Add your assets:

- Put images, PDFs, and/or video clips in `data/slides/`
- Edit `data/texts.txt`

Narration entries are separated with a single `-` line:

```text
Welcome to GoCreator
-
This is the second slide
-
This is the third slide
```

Create videos:

```bash
gocreator create --lang en --langs-out en,fr,es
```

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

then `data/texts.txt` must contain 5 narration entries:

1. `01-cover.png`
2. `02-handout.pdf` page 1
3. `02-handout.pdf` page 2
4. `02-handout.pdf` page 3
5. `03-demo.mp4`

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

1. Load narration from `data/texts.txt`
2. Load slides from `data/slides`
3. Expand and cache PDF pages when needed
4. Translate narration for non-source languages
5. Generate narration audio
6. Render one video segment per final slide
7. Concatenate segments into `data/out/output-<lang>.mp4`

## Configuration notes

The config schema is larger than the currently wired runtime.

The main `create` flow actively uses:

- `input.lang`
- `output.languages`
- `cache`
- `transition`
- `multi_view`

Other config sections remain in the schema, but they are not yet part of the core `create` pipeline.

## Examples

- `examples/getting-started/` - minimal local workflow
- `examples/demo/` - end-to-end CLI example
- `examples/demo-multiview/` - multi-view example
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
