# Cache Management Policy

This document describes the caching strategy used in gocreator to optimize performance and reduce API costs.

## Overview

The application implements a multi-layered caching strategy to avoid redundant API calls and ffmpeg operations:

1. **Translation Cache** - File-based cache for translated texts
2. **Audio Generation Cache** - File-based cache with hash validation for generated audio
3. **Video Segment Cache** - Automatic caching of individual video segments
4. **In-Memory Cache** - General-purpose cache service for runtime data

## 1. Translation Cache

**Location**: `data/cache/{language}/text/texts.txt`

**Purpose**: Avoid re-translating the same texts when re-running the tool

**Strategy**:
- When translating to a target language, the service first checks if a cached translation file exists
- If found, it loads the cached translations instead of calling the OpenAI API
- If not found, it translates the texts and saves them to the cache file

**Cache Key**: Language code (e.g., "es", "fr", "de")

**Expiration**: **Never expires** - Filesystem cache persists indefinitely

**Invalidation**: 
- Manual deletion of cache files
- When source texts change (detected by content comparison)

## 2. Audio Generation Cache

**Location**: `data/cache/{language}/audio/{index}.mp3` and corresponding `.hash` files

**Purpose**: Avoid regenerating audio for the same text content

**Strategy**:
- Before generating audio, the service computes a SHA256 hash of the text
- It checks if an audio file with a matching hash already exists
- If cached, it reuses the existing file
- If not, it generates new audio and saves both the audio file and its hash

**Cache Key**: SHA256 hash of the input text

**Hash Files**: Each audio file has a corresponding `.hash` file containing the SHA256 hash of the text that generated it

**Expiration**: **Never expires** - Filesystem cache persists indefinitely

**Invalidation**: 
- Automatic when text content changes (hash mismatch detected via SHA256)
- Manual deletion of audio files or hash files

## 3. Video Segment Cache

**Location**: `data/out/.temp/video_{index}.mp4`

**Purpose**: Cache individual video segments before concatenation

**Strategy**:
- Each slide+audio combination is rendered as a separate video segment
- Segments are generated in parallel for performance
- All segments are then concatenated into the final video

**Expiration**: **Never expires** - Filesystem cache persists indefinitely

**Benefits**:
- Enables parallel processing of segments
- Allows reuse of segments if only some slides change
- Simplifies debugging and inspection of individual segments
- No time-based expiration - only hash-based invalidation

## 4. In-Memory Cache Service

**Implementation**: `internal/services/cache.go`

**Purpose**: General-purpose caching for runtime data that doesn't need persistence

**Features**:
- Time-based expiration (TTL)
- Automatic cleanup of expired entries
- Type-agnostic storage (stores interface{})

**Expiration**: **TTL-based** - Entries expire after configured duration

**Usage**: Can be used by services for temporary runtime caching needs

**Important**: This is the ONLY cache type with TTL expiration. All filesystem-based caches (translations, audio, video segments) persist indefinitely and use hash-based invalidation instead of time-based expiration.

## Cache Directory Structure

```
data/
├── cache/
│   ├── en/                    # English (input language)
│   │   ├── text/
│   │   │   └── texts.txt      # Not cached (same as input)
│   │   └── audio/
│   │       ├── 0.mp3
│   │       ├── 0.mp3.hash
│   │       ├── 1.mp3
│   │       ├── 1.mp3.hash
│   │       └── hashes         # Index of all hashes
│   ├── es/                    # Spanish translation
│   │   ├── text/
│   │   │   └── texts.txt      # Cached translation
│   │   └── audio/
│   │       ├── 0.mp3
│   │       ├── 0.mp3.hash
│   │       └── ...
│   └── fr/                    # French translation
│       └── ...
└── out/
    ├── .temp/                 # Temporary video segments
    │   ├── video_0.mp4
    │   ├── video_1.mp4
    │   └── ...
    ├── output-en.mp4          # Final videos
    ├── output-es.mp4
    └── output-fr.mp4
```

## Performance Implications

### First Run
- All translations are performed via OpenAI API
- All audio is generated via OpenAI TTS API
- All video segments are rendered via ffmpeg

### Subsequent Runs (same texts)
- ✅ Translations are loaded from cache (no API calls)
- ✅ Audio files are reused (no API calls)
- ⚠️ Video segments are regenerated (ffmpeg re-runs)

### Subsequent Runs (modified texts)
- ⚠️ Modified texts are re-translated (API calls for changed items)
- ⚠️ Modified audio is regenerated (API calls for changed items)
- ✅ Unchanged texts/audio are reused from cache

## Best Practices

1. **Preserve Cache Directory**: Keep the `data/cache` directory between runs to benefit from caching

2. **Monitor Hash Files**: Ensure `.hash` files are preserved alongside `.mp3` files

3. **Clean Old Caches**: Periodically clean up cache directories for languages you no longer need

4. **Version Control**: Add `data/cache/` and `data/out/.temp/` to `.gitignore`

## Future Enhancements

Potential improvements to the caching system:

1. **FFmpeg Output Caching**: Cache rendered video segments based on slide+audio hash
2. **Cache Compression**: Compress cache files to save disk space
3. **Cache Eviction Policy**: Implement LRU eviction for disk caches
4. **Cache Statistics**: Add logging for cache hit/miss rates
5. **Distributed Cache**: Support for shared cache across multiple machines
