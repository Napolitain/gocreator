# Performance Testing Tool

This tool provides comprehensive performance testing for GoCreator's core functions, including cache hit tracking and end-to-end latency measurements.

## Features

- **Benchmark Tests**: Standard Go benchmark tests for all core services
- **Performance Testing Tool**: Standalone command for end-to-end performance analysis
- **Cache Hit Tracking**: Logs and counts cache hits during operations
- **Markdown Output**: Generates formatted tables for easy performance comparison

## Running Benchmark Tests

Use the standard Go benchmark testing:

```bash
# Run all benchmarks
go test -bench=. ./internal/services/ -run=^$

# Run specific benchmark
go test -bench=BenchmarkAudioService ./internal/services/ -run=^$

# Run with longer duration for more accurate results
go test -bench=. -benchtime=5s ./internal/services/ -run=^$

# Run with memory allocation statistics
go test -bench=. -benchmem ./internal/services/ -run=^$
```

### Benchmark Categories

1. **Audio Service Benchmarks**
   - `BenchmarkAudioService_Generate_NoCache` - Audio generation without cache
   - `BenchmarkAudioService_Generate_WithCache` - Audio generation with cache hit
   - `BenchmarkAudioService_GenerateBatch_NoCache` - Batch generation without cache
   - `BenchmarkAudioService_GenerateBatch_WithCache` - Batch generation with cache

2. **Translation Service Benchmarks**
   - `BenchmarkTranslationService_Translate` - Single translation
   - `BenchmarkTranslationService_TranslateBatch_*Texts` - Batch translations with varying sizes

3. **Cache Service Benchmarks**
   - `BenchmarkCacheService_Set` - Cache write operations
   - `BenchmarkCacheService_Get_Hit` - Cache read with hit
   - `BenchmarkCacheService_Get_Miss` - Cache read with miss
   - `BenchmarkCacheService_Delete` - Cache deletion
   - `BenchmarkCacheService_MixedOperations` - Mixed read/write patterns

4. **Text Service Benchmarks**
   - `BenchmarkTextService_Load` - Load text from file
   - `BenchmarkTextService_Save` - Save text to file
   - `BenchmarkTextService_Hash` - Hash computation
   - `BenchmarkTextService_LoadHashes` - Hash file loading
   - `BenchmarkTextService_SaveHashes` - Hash file saving

## Running the Performance Testing Tool

The `perftest` tool provides end-to-end performance testing with real or simulated API calls.

### Build and Run

```bash
# Build the tool
go build -o perftest ./cmd/perftest/

# Run in simulation mode (no API key required)
./perftest

# Run with real OpenAI API (requires OPENAI_API_KEY)
export OPENAI_API_KEY="your-api-key-here"
./perftest
```

### Output

The tool generates a markdown-formatted table with performance metrics:

```
Performance Test Results
========================

| Operation | Cache Status | Total Duration | Iterations | Avg Duration |
|-----------|--------------|----------------|------------|--------------|
| Text Load                      | N/A          | 10.189055ms    | 1000 | 10.189µs |
| Text Hash                      | N/A          | 237.847µs     | 1000 | 237ns |
| Cache Set                      | N/A          | 3.029927ms     | 10000 | 302ns |
| Cache Get (hit)                | Hit          | 1.689887ms     | 10000 | 168ns |
| Cache Get (miss)               | Miss         | 1.238899ms     | 10000 | 123ns |

End-to-End Performance:
| E2E Without Cache | 15.2s |
| E2E With Cache    | 1.3s |
| Speedup Factor    | 11.69x |
| Cache Hit Count   | 6 / 6 operations |
```

## Simulation vs Real API Mode

### Simulation Mode (Default)

When `OPENAI_API_KEY` is not set, the tool runs in simulation mode:

- Tests text processing operations (1000 iterations)
- Tests cache operations (10000 iterations)
- Uses mocked responses for fast testing
- No API costs incurred

### Real API Mode

When `OPENAI_API_KEY` is set, the tool performs actual API calls:

- Tests translation API with 3 sample texts
- Tests audio generation API with 3 sample texts
- Measures cache hit rates
- Calculates end-to-end latency improvements
- **Note**: This will incur OpenAI API costs

## Understanding the Results

### Key Metrics

1. **Total Duration**: Total time for all iterations
2. **Iterations**: Number of times the operation was performed
3. **Avg Duration**: Average time per operation (Total / Iterations)
4. **Cache Status**: Whether the operation used cache (Hit/Miss/No Cache/N/A)

### Cache Performance

The benchmarks demonstrate the performance impact of caching:

- **Audio Generation**: ~12x faster with cache (without cache: ~14.4µs, with cache: ~1.2µs)
- **Cache Operations**: Get operations are ~3x faster than Set operations
- **Text Operations**: Hash computation is very fast (~257ns per operation)

### End-to-End Performance

When running with real API:

- Translation and audio generation are measured separately
- Cache hits are logged and counted
- Speedup factor shows the performance improvement from caching
- Typical speedup is 10-15x for cached operations

## Best Practices

1. **Run benchmarks multiple times** to account for variability
2. **Use `-benchtime=5s`** for more stable results
3. **Monitor cache hit rates** to optimize caching strategy
4. **Test with realistic data sizes** matching your production workload
5. **Compare results before/after changes** to measure impact

## Interpreting Benchmark Results

Example benchmark output:

```
BenchmarkAudioService_Generate_NoCache-4    80286    14420 ns/op
BenchmarkAudioService_Generate_WithCache-4  957981    1215 ns/op
```

This means:
- Without cache: 80,286 operations in 1 second, ~14.4µs per operation
- With cache: 957,981 operations in 1 second, ~1.2µs per operation
- Cache provides ~12x performance improvement

## Troubleshooting

### Benchmark Variability

If benchmark results vary significantly:
- Increase `-benchtime` (e.g., `-benchtime=10s`)
- Run benchmarks multiple times and average results
- Ensure system is not under heavy load during testing

### API Rate Limits

When testing with real API:
- OpenAI has rate limits - the tool uses only 3 texts to stay within limits
- If you hit rate limits, wait and try again
- Consider using simulation mode for frequent testing

## Contributing

When adding new services or features:

1. Add corresponding benchmark tests in `internal/services/*_benchmark_test.go`
2. Follow the naming convention: `Benchmark<Service>_<Operation>_<Condition>`
3. Test both cached and non-cached scenarios where applicable
4. Update this README with new benchmark descriptions

## Related Documentation

- [CACHE_POLICY.md](../../CACHE_POLICY.md) - Detailed caching strategy
- [README.md](../../README.md) - Main project documentation
- [Go Benchmark Documentation](https://pkg.go.dev/testing#hdr-Benchmarks) - Official Go testing docs
