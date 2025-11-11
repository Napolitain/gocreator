# GoCreator - Video Creation Tool

[![CI](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml/badge.svg)](https://github.com/Napolitain/gocreator/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Napolitain/gocreator/branch/main/graph/badge.svg)](https://codecov.io/gh/Napolitain/gocreator)
[![Go Report Card](https://goreportcard.com/badge/github.com/Napolitain/gocreator)](https://goreportcard.com/report/github.com/Napolitain/gocreator)

A CLI tool for creating videos with translations and audio narration.

> 📋 **Roadmap**: See our [Roadmap Summary](./ROADMAP_SUMMARY.md) or [Full Improvement Plan](./IMPROVEMENTS_ROADMAP.md) for future features and enhancements.

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/Napolitain/gocreator/releases).

```bash
# Linux/macOS
chmod +x gocreator-*
sudo mv gocreator-* /usr/local/bin/gocreator

# Verify installation
gocreator --help
```

### From Source

```bash
go install github.com/Napolitain/gocreator/cmd/gocreator@latest
```

## Features

- Automated video creation from slides and text
- **Video input support** - Use video clips as "slides" with their duration, not just static images
- **Google Slides API integration** - Fetch slides and speaker notes directly from Google Slides
- Multi-language support with AI-powered translation
- Text-to-speech audio generation
- Intelligent caching to reduce API costs
- Parallel processing for better performance

## Quick Start

**New to GoCreator?** Check out the [examples/](./examples/) directory for a hands-on tutorial:

```bash
cd examples/getting-started
gocreator create --lang en --langs-out en,fr,es
```

See the [Getting Started Example](./examples/getting-started/) for detailed instructions.

## Usage

### Using Local Slides

Create a `data` directory in your project with:
- `data/slides/` - Directory containing slide images (PNG, JPEG) or video clips (MP4, MOV, AVI, MKV, WEBM)
- `data/texts.txt` - Text file with slide narrations separated by `-`

```bash
gocreator create --lang en --langs-out en,fr,es
```

**How it works**:
- **Image slides**: Duration is determined by the TTS audio length
- **Video slides**: Duration is determined by the video length, with TTS audio aligned at the beginning
- You can mix images and videos in the same presentation

### Using Google Slides

GoCreator supports **two authentication methods** for Google Slides API:

**Option A: OAuth 2.0 (Personal Use)**
1. **Set up Google Cloud Project** and enable Google Slides API
2. **Create OAuth 2.0 credentials** for desktop app
3. **Set environment variable**: `export GOOGLE_OAUTH_CREDENTIALS="/path/to/oauth-credentials.json"`
4. **Run with Google Slides**: `gocreator create --google-slides YOUR_PRESENTATION_ID --lang en --langs-out en,fr,es`
5. **Authorize on first run**: Follow the prompts to authorize with your Google account

**Option B: Service Account (CI/CD)**
1. **Set up Google Cloud Project** and enable Google Slides API
2. **Create service account credentials** and download JSON file
3. **Share your presentation** with the service account email
4. **Set environment variable**: `export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"`
5. **Run with Google Slides**: `gocreator create --google-slides YOUR_PRESENTATION_ID --lang en --langs-out en,fr,es`

The presentation ID can be found in the Google Slides URL:
```
https://docs.google.com/presentation/d/[PRESENTATION_ID]/edit
```

**How it works**:
- Slides are downloaded as images from your Google Slides presentation
- Speaker notes from each slide are used as the narration text
- Videos are generated with audio in multiple languages
- All content is cached for efficient re-generation
- OAuth 2.0 provides automatic token refresh for seamless access

📖 **See [GOOGLE_SLIDES_GUIDE.md](GOOGLE_SLIDES_GUIDE.md) for detailed setup instructions and troubleshooting.**

## Versioning

This project uses **Calendar Versioning (CalVer)** with the format `YYYY-MM-DD`.

Each release is tagged with the date it was created (e.g., `2025-01-15`). This makes it easy to:
- Know when a version was released
- Track the age of your installation
- Plan upgrades based on release frequency

## Architecture

The project follows clean architecture principles with clear separation of concerns:

### Layers

1. **CLI Layer** (`internal/cli/`)
   - Command-line interface and user interaction
   - Minimal business logic
   
2. **Service Layer** (`internal/services/`)
   - Business logic and orchestration
   - VideoCreator orchestrates the entire video creation workflow
   - Individual services handle specific concerns (text, audio, video, translation)

3. **Adapter Layer** (`internal/adapters/`)
   - External API integrations (OpenAI)
   - Wraps third-party clients with our interfaces

4. **Interface Layer** (`internal/interfaces/`)
   - Defines contracts between layers
   - Enables dependency injection and testing

### Dependency Injection

All services follow dependency injection principles:

```go
// Services receive dependencies through constructors
textService := services.NewTextService(fs, logger)
audioService := services.NewAudioService(fs, openaiClient, textService, logger)

// VideoCreator depends on interfaces, not concrete types
creator := services.NewVideoCreator(
    fs,
    textService,    // interfaces.TextProcessor
    translation,    // interfaces.Translator
    audioService,   // interfaces.AudioGenerator
    videoService,   // interfaces.VideoGenerator
    slideService,   // interfaces.SlideLoader
    logger,         // interfaces.Logger
)
```

This design enables:
- Easy testing with mocks
- Swapping implementations without changing code
- Clear dependency graph
- Better maintainability

## Testing

The project includes comprehensive unit tests with mocked dependencies:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/services/...

# Run benchmark tests
go test -bench=. ./internal/services/ -run=^$

# Run benchmarks with memory stats
go test -bench=. -benchmem ./internal/services/ -run=^$
```

### Performance Testing

GoCreator includes a comprehensive performance testing tool that measures cache performance, API latency, and provides end-to-end metrics:

```bash
# Build the performance testing tool
go build -o perftest ./cmd/perftest/

# Run in simulation mode (no API key needed)
./perftest

# Run with real OpenAI API (requires OPENAI_API_KEY)
export OPENAI_API_KEY="your-key"
./perftest
```

The tool generates markdown-formatted performance tables showing:
- Operation timings with and without cache
- Cache hit rates and counts
- End-to-end latency measurements
- Performance improvement factors

See [cmd/perftest/README.md](cmd/perftest/README.md) for detailed documentation.

### Test Coverage

- **TextService**: Load, Save, Hash, and hash file operations
- **AudioService**: Audio generation with cache validation
- **TranslationService**: Single and batch translations
- **CacheService**: Set, Get, Delete, Clear, Expiration
- **VideoCreator**: Full workflow orchestration with mocked services
- **Benchmarks**: Performance tests for all core operations with cache scenarios

All external dependencies (filesystem, OpenAI API) are mocked for isolated unit testing.

## Cache Management

GoCreator implements a sophisticated multi-layered caching strategy:

### Cache Types

1. **Translation Cache** - Saves API costs by caching translations
2. **Audio Cache** - Reuses generated audio with hash validation
3. **Video Segment Cache** - Caches intermediate video segments
4. **In-Memory Cache** - Runtime caching with TTL support

See [CACHE_POLICY.md](CACHE_POLICY.md) for detailed documentation.

### Cache Benefits

- **Cost Reduction**: Avoids redundant OpenAI API calls
- **Performance**: Reuses expensive computations
- **Reliability**: Works offline for previously processed content
- **Debugging**: Easier to inspect cached intermediate results

## Development

### Project Structure

```
gocreator/
├── cmd/
│   └── gocreator/          # Main entry point
├── internal/
│   ├── adapters/           # External API adapters
│   ├── cli/                # CLI commands
│   ├── interfaces/         # Interface definitions
│   ├── mocks/             # Mock implementations for testing
│   └── services/          # Business logic
│       ├── audio.go       # Audio generation
│       ├── cache.go       # Cache management
│       ├── creator.go     # Main orchestrator
│       ├── slide.go       # Slide loading
│       ├── text.go        # Text processing
│       ├── translation.go # Translation service
│       ├── video.go       # Video generation
│       └── *_test.go      # Unit tests
├── CACHE_POLICY.md        # Cache strategy documentation
└── go.mod
```

### Adding New Features

To add a new service:

1. Define the interface in `internal/interfaces/interfaces.go`
2. Implement the service in `internal/services/`
3. Create mock in `internal/mocks/` for testing
4. Write comprehensive unit tests
5. Update VideoCreator to use the new service

Example:
```go
// 1. Define interface
type SubtitleGenerator interface {
    Generate(ctx context.Context, texts []string, outputPath string) error
}

// 2. Implement service
type SubtitleService struct {
    fs     afero.Fs
    logger interfaces.Logger
}

func NewSubtitleService(fs afero.Fs, logger interfaces.Logger) *SubtitleService {
    return &SubtitleService{fs: fs, logger: logger}
}

// 3. Create mock (in internal/mocks/)
type MockSubtitleGenerator struct {
    mock.Mock
}

// 4. Write tests
func TestSubtitleService_Generate(t *testing.T) {
    // ...
}
```

## Dependencies

- **github.com/spf13/cobra** - CLI framework
- **github.com/spf13/afero** - Filesystem abstraction (enables easy testing)
- **github.com/openai/openai-go** - OpenAI API client
- **github.com/patrickmn/go-cache** - In-memory cache with expiration
- **github.com/stretchr/testify** - Testing framework with mocking support

## Best Practices

### Code Quality

1. **Interfaces over Concrete Types**: Services depend on interfaces for flexibility
2. **Dependency Injection**: All dependencies passed through constructors
3. **Single Responsibility**: Each service has one clear purpose
4. **Comprehensive Testing**: Mock external dependencies for unit tests
5. **Error Handling**: Clear error messages with context

### Testing Strategy

1. **Unit Tests**: Test each service in isolation with mocks
2. **Table-Driven Tests**: Use test tables for multiple scenarios
3. **Mock External APIs**: Never call real APIs in unit tests
4. **Test Edge Cases**: Empty inputs, errors, concurrent operations

### Caching Strategy

1. **Hash-Based Validation**: Use content hashes to detect changes
2. **Layered Caching**: Multiple cache levels for different concerns
3. **Cache Invalidation**: Automatic invalidation on content changes
4. **Documented Policy**: Clear documentation of cache behavior

## Future Improvements

> **📋 Full Roadmap**: See [IMPROVEMENTS_ROADMAP.md](./IMPROVEMENTS_ROADMAP.md) for a comprehensive improvement plan covering usability, features, platforms, and technical enhancements.

Key areas for future development:

### High Priority
1. **Subtitle Support**: Automatic subtitle generation in multiple formats (SRT, VTT)
2. **Configuration Files**: YAML/JSON config for easier project management
3. **Multiple Voice Options**: Choose from different TTS voices and providers
4. **Better Error Messages**: User-friendly errors with actionable solutions
5. **Progress Indicators**: Rich progress bars and status updates

### Medium Priority
6. **PowerPoint Support**: Direct .pptx file support
7. **Resume Support**: Resume interrupted video creation jobs
8. **Quality Profiles**: Preset quality configurations (low, medium, high, ultra)
9. **Dry Run Mode**: Preview costs and outputs before processing
10. **Alternative TTS Providers**: Support for Google, AWS, Azure, ElevenLabs

### Also Planned
- Background music support
- Video transitions and effects
- YouTube/social media integration
- Batch processing
- Plugin system for extensibility
- API and webhooks
- Advanced caching strategies

## Contributing

When contributing:

1. Follow existing code structure and patterns
2. Add unit tests for all new code
3. Update documentation for user-facing changes
4. Keep functions small and focused
5. Use meaningful variable and function names
6. Add comments for complex logic only

## License

See [LICENSE](LICENSE) file for details.
