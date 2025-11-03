# Getting Started Example

This example demonstrates how to use GoCreator to create a simple multi-language video presentation.

## What's Included

- **5 sample slides** (`data/slides/`) - Colorful slides introducing GoCreator
- **Narration text** (`data/texts.txt`) - Speaker notes for each slide, separated by `-`

## Structure

```
getting-started/
├── README.md           # This file
└── data/
    ├── slides/         # Slide images (PNG format)
    │   ├── slide1.png
    │   ├── slide2.png
    │   ├── slide3.png
    │   ├── slide4.png
    │   └── slide5.png
    └── texts.txt       # Narration text for each slide
```

## How to Run

### Prerequisites

1. **Install GoCreator** - Follow the installation instructions in the main README
2. **Set up OpenAI API** - Set your API key:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

### Running the Example

Navigate to this example directory and run:

```bash
cd examples/getting-started
gocreator create --lang en --langs-out en,fr,es
```

This command will:
1. Load the 5 slides from `data/slides/`
2. Load the narration text from `data/texts.txt`
3. Generate videos in English, French, and Spanish
4. Create output videos in `data/output/`

### Understanding the Command

- `--lang en` - Specifies that the input text is in English
- `--langs-out en,fr,es` - Generate videos in English, French, and Spanish

### Expected Output

After running the command, you'll find:

```
data/
├── output/
│   ├── en/
│   │   └── output.mp4       # English video
│   ├── fr/
│   │   └── output.mp4       # French video
│   └── es/
│       └── output.mp4       # Spanish video
├── translations/            # Cached translations
├── audio/                   # Cached audio files
└── segments/                # Cached video segments
```

## Customizing the Example

### Changing the Slides

Replace the PNG files in `data/slides/` with your own images. Supported formats:
- Images: PNG, JPG, JPEG
- Videos: MP4, MOV, AVI, MKV, WEBM

**Note**: Number of slides must match the number of text sections in `texts.txt`.

### Editing the Narration

Edit `data/texts.txt` to change the narration. Use a single `-` on its own line to separate text for different slides:

```
Text for slide 1
-
Text for slide 2
-
Text for slide 3
```

### Adding More Languages

To generate videos in additional languages, add them to the `--langs-out` parameter:

```bash
gocreator create --lang en --langs-out en,fr,es,de,ja
```

Supported languages include: en, fr, es, de, ja, zh, pt, ru, it, ar, and many more.

## How It Works

1. **Text Processing**: Loads and parses `texts.txt`, splitting by `-` delimiter
2. **Translation**: Translates text to each target language using OpenAI API
3. **Audio Generation**: Generates natural-sounding speech for each language
4. **Video Assembly**: Combines slides with audio to create final videos
5. **Caching**: Saves translations and audio to avoid redundant API calls

## Tips

- **First run will be slower** - Subsequent runs use cached translations and audio
- **Check your API costs** - Each run makes OpenAI API calls for translation and TTS
- **Use caching effectively** - Don't change text unnecessarily to maximize cache hits
- **Mix images and videos** - You can use both image slides and video clips

## Troubleshooting

### "slide and text count mismatch"
- Ensure the number of slides matches the number of text sections in `texts.txt`
- Count the `-` delimiters: 5 slides need 4 delimiters (5 sections)

### "failed to open file: data/texts.txt"
- Make sure you're running the command from the `examples/getting-started` directory
- Or use absolute paths in your configuration

### "OpenAI API error"
- Verify your `OPENAI_API_KEY` environment variable is set
- Check your OpenAI account has sufficient credits
- Ensure your API key has access to the required models

## Next Steps

- Try creating your own slides and narration
- Explore the [Google Slides integration](../../GOOGLE_SLIDES_GUIDE.md)
- Read about [caching strategies](../../CACHE_POLICY.md)
- Check the main [README](../../README.md) for advanced features

## Learn More

- GitHub Repository: https://github.com/Napolitain/gocreator
- Report Issues: https://github.com/Napolitain/gocreator/issues
- Documentation: See the main README for detailed information
