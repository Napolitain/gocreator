# Google Slides Integration Guide

This guide explains how to set up and use Google Slides API with GoCreator.

## Overview

GoCreator can fetch slides and speaker notes directly from Google Slides presentations. This eliminates the need to manually export slides as images and allows you to keep your content in sync with your Google Slides presentation.

## Features

- **Automatic Slide Export**: Slides are automatically downloaded as images
- **Speaker Notes as Narration**: Speaker notes from each slide become the narration text
- **Live Updates**: Re-run to fetch the latest version of your presentation
- **Caching**: Downloaded slides and generated videos are cached for efficiency

## Setup Instructions

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Create Project" or select an existing project
3. Note your Project ID

### 2. Enable Google Slides API

1. In the Google Cloud Console, navigate to "APIs & Services" > "Library"
2. Search for "Google Slides API"
3. Click on it and click "Enable"

### 3. Create Service Account Credentials

For automated access (recommended for CI/CD):

1. Navigate to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "Service Account"
3. Enter a name (e.g., "gocreator-service")
4. Click "Create and Continue"
5. Skip granting access (optional)
6. Click "Done"
7. Click on the created service account
8. Go to "Keys" tab
9. Click "Add Key" > "Create new key"
10. Choose "JSON" format
11. Download the credentials file
12. Save it securely (e.g., `~/.config/gocreator/credentials.json`)

### 4. Share Your Google Slides Presentation

**Important**: You must share your Google Slides presentation with the service account email address.

1. Open your Google Slides presentation
2. Click the "Share" button
3. Add the service account email (found in the credentials JSON file, looks like `gocreator-service@your-project.iam.gserviceaccount.com`)
4. Give it "Viewer" permission
5. Click "Send"

### 5. Set Environment Variable

Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to point to your credentials file:

**Linux/macOS:**
```bash
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gocreator/credentials.json"
```

**Windows (PowerShell):**
```powershell
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\Users\YourName\.config\gocreator\credentials.json"
```

**Persistent setup (add to ~/.bashrc or ~/.zshrc):**
```bash
echo 'export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gocreator/credentials.json"' >> ~/.bashrc
source ~/.bashrc
```

## Usage

### Find Your Presentation ID

The presentation ID is in the URL of your Google Slides:

```
https://docs.google.com/presentation/d/1ABC-xyz123_EXAMPLE-ID/edit
                                    └─────────────┬─────────────┘
                                            Presentation ID
```

### Run GoCreator

```bash
# Single language
gocreator create --google-slides 1ABC-xyz123_EXAMPLE-ID --lang en

# Multiple languages
gocreator create --google-slides 1ABC-xyz123_EXAMPLE-ID --lang en --langs-out en,fr,es,de
```

### What Happens

1. GoCreator connects to Google Slides API using your credentials
2. Fetches the presentation metadata
3. Downloads each slide as a PNG image
4. Extracts speaker notes from each slide
5. Uses the notes as narration text for video generation
6. Generates videos with audio in all specified languages

## Best Practices

### 1. Speaker Notes

- **Be descriptive**: Write clear, complete sentences in speaker notes
- **One slide, one topic**: Keep each slide's narration focused
- **Timing**: Aim for 30-60 seconds of narration per slide
- **Language**: Write notes in your input language (specified with `--lang`)

### 2. Slide Design

- **Keep it simple**: Simple slides work best for video
- **High contrast**: Ensure text is readable
- **Avoid animations**: Slides are exported as static images
- **Consistent size**: Use a standard slide size (16:9 recommended)

### 3. Workflow

```bash
# 1. Create/update your Google Slides presentation
# 2. Add speaker notes to each slide
# 3. Run gocreator
gocreator create --google-slides YOUR_ID --lang en --langs-out en,fr,es

# 4. Check output in data/out/
ls data/out/
# output-en.mp4  output-fr.mp4  output-es.mp4
```

### 4. Caching

- Slides are cached in `data/slides/`
- Notes are saved to `data/texts.txt`
- To refresh from Google Slides, delete these and re-run
- Translations and audio are cached separately

## Troubleshooting

### Error: "GOOGLE_APPLICATION_CREDENTIALS environment variable not set"

**Solution**: Set the environment variable as described in step 5 above.

### Error: "failed to get presentation: googleapi: Error 404"

**Possible causes**:
1. Incorrect presentation ID
2. Presentation not shared with service account
3. Presentation deleted

**Solution**: 
- Verify the presentation ID
- Check that you shared the presentation with the service account email
- Ensure the presentation exists

### Error: "failed to create slides service: ... credentials"

**Possible causes**:
1. Credentials file not found
2. Invalid credentials file
3. Wrong file path in environment variable

**Solution**:
- Check that the credentials file exists at the specified path
- Verify the credentials file is valid JSON
- Check the environment variable is set correctly

### Error: "failed to get thumbnail"

**Possible causes**:
1. Google Slides API not enabled
2. API quota exceeded
3. Network issues

**Solution**:
- Ensure Google Slides API is enabled in your project
- Check API usage quotas in Google Cloud Console
- Verify network connectivity

## Advanced Usage

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Generate Videos
on:
  push:
    branches: [main]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup credentials
        env:
          GOOGLE_CREDENTIALS: ${{ secrets.GOOGLE_CREDENTIALS }}
        run: |
          echo "$GOOGLE_CREDENTIALS" > credentials.json
          echo "GOOGLE_APPLICATION_CREDENTIALS=$PWD/credentials.json" >> $GITHUB_ENV
      
      - name: Install gocreator
        run: |
          # Download and install gocreator
          
      - name: Generate videos
        run: |
          gocreator create --google-slides ${{ secrets.PRESENTATION_ID }} --lang en --langs-out en,fr
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: videos
          path: data/out/*.mp4
```

### Multiple Presentations

Process multiple presentations by running gocreator multiple times with different presentation IDs:

```bash
for id in "PRES_ID_1" "PRES_ID_2" "PRES_ID_3"; do
  gocreator create --google-slides "$id" --lang en --langs-out en,fr
  mv data/out/*.mp4 "output/$id/"
done
```

## Security Notes

1. **Keep credentials secure**: Never commit credentials to version control
2. **Use .gitignore**: Add `credentials.json` to your `.gitignore`
3. **Limit permissions**: Service account only needs "Viewer" access
4. **Rotate credentials**: Periodically create new service account keys
5. **Use secrets managers**: In production, use Google Secret Manager or similar

## Support

If you encounter issues:

1. Check this guide's troubleshooting section
2. Verify your Google Cloud project setup
3. Check API quotas and limits
4. Open an issue on GitHub with:
   - Error message (remove sensitive info)
   - Steps to reproduce
   - GoCreator version
   - OS and environment details
