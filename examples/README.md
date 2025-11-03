# GoCreator Examples

This directory contains example projects demonstrating how to use GoCreator.

## Available Examples

### [getting-started](./getting-started/)

A simple introductory example that demonstrates the basic usage of GoCreator.

**What you'll learn:**
- How to structure a GoCreator project
- Creating slides and narration text
- Running the CLI with basic options
- Generating multi-language videos

**Complexity:** Beginner  
**Time to complete:** 5-10 minutes  
**Prerequisites:** OpenAI API key

## Running Examples

Each example contains its own README with detailed instructions. General steps:

1. Navigate to the example directory:
   ```bash
   cd examples/getting-started
   ```

2. Ensure you have the required credentials:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

3. Run the example:
   ```bash
   gocreator create --lang en --langs-out en,fr,es
   ```

## Example Structure

Each example follows this structure:

```
example-name/
├── README.md           # Instructions and documentation
└── data/
    ├── slides/         # Slide images or videos
    └── texts.txt       # Narration text
```

## Prerequisites for All Examples

1. **GoCreator installed** - See main [README](../README.md) for installation
2. **OpenAI API key** - Required for translation and text-to-speech
3. **FFmpeg** - Used for video processing (usually pre-installed)

## Contributing Examples

We welcome contributions! To add a new example:

1. Create a new directory under `examples/`
2. Follow the standard structure (data/slides/, data/texts.txt)
3. Include a comprehensive README.md
4. Test your example thoroughly
5. Submit a pull request

Good examples should:
- Demonstrate a specific feature or use case
- Include clear documentation
- Work out of the box
- Be educational and practical

## Additional Resources

- [Main Documentation](../README.md)
- [Google Slides Guide](../GOOGLE_SLIDES_GUIDE.md)
- [Cache Policy](../CACHE_POLICY.md)
- [GitHub Repository](https://github.com/Napolitain/gocreator)
