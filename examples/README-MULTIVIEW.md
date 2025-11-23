# Multi-View Examples - Getting Started

## 📁 **About These Examples**

The multi-view example YAML files show **configuration structure** but reference video files that you need to provide.

---

## 🎬 **What You Need**

To use these examples, you'll need:

### **Video Files**
The examples reference video files like:
- `videos/intro-background.mp4`
- `videos/host-intro.mp4`
- `videos/screen-demo.mp4`
- `videos/instructor.mp4`
- etc.

**You need to provide these video files yourself** based on your use case.

---

## 🚀 **Quick Start**

### **Option 1: Use Your Own Videos**

1. **Create a videos directory**:
   ```bash
   mkdir -p data/videos
   ```

2. **Add your video files**:
   ```
   data/videos/
   ├── main.mp4        # Your main content
   ├── webcam.mp4      # Your webcam recording
   └── screen.mp4      # Your screen recording
   ```

3. **Update the YAML** to reference your files:
   ```yaml
   multi_view:
     enabled: true
     layouts:
       - type: split-horizontal
         slides: all
         ratio: 60:40
         videos:
           left: data/videos/screen.mp4
           right: data/videos/webcam.mp4
   ```

---

### **Option 2: Simple Tutorial Example**

Here's a minimal working example you can adapt:

**File: `my-tutorial.yaml`**

```yaml
input:
  lang: en
  source: local

output:
  languages: [en]
  directory: ./data/out

# Simple split-screen tutorial
multi_view:
  enabled: true
  layouts:
    - type: split-horizontal
      slides: all  # Apply to all slides
      ratio: 65:35
      videos:
        left: ./my-screen-recording.mp4
        right: ./my-webcam.mp4
      gap: 4

encoding:
  video:
    codec: libx264
    preset: medium
  audio:
    codec: aac
    bitrate: 192k
```

**What you need**:
1. ✅ Text file: `data/texts.txt` (your narration)
2. ✅ Slides: `data/slides/` (your slide images)
3. ✅ Video files: `my-screen-recording.mp4` and `my-webcam.mp4`

Then run:
```bash
gocreator create --config my-tutorial.yaml
```

---

### **Option 3: Gaming Example**

**File: `my-gaming.yaml`**

```yaml
input:
  lang: en
  source: local

output:
  languages: [en]
  directory: ./data/out

# Gaming with facecam overlay
multi_view:
  enabled: true
  layouts:
    - type: pip
      slides: all
      main: ./gameplay.mp4
      overlay: ./facecam.mp4
      position: bottom-right
      size: 20%
      border:
        width: 2
        color: "#00FF00"

encoding:
  video:
    codec: libx264
    preset: fast
```

**What you need**:
1. ✅ Gameplay recording: `gameplay.mp4`
2. ✅ Facecam recording: `facecam.mp4`
3. ✅ Text file: `data/texts.txt`
4. ✅ Slides: `data/slides/`

---

## 📝 **Example Explanations**

### **1. multiview-demo.yaml**
Shows all 5 layout types. **Reference only** - update paths to your videos.

### **2. multiview-tutorial.yaml**
Simple split-screen tutorial. **Update video paths** before using.

### **3. multiview-interview.yaml**
Interview setup with 50:50 split. **Update video paths** before using.

### **4. multiview-gaming.yaml**
Gaming with PiP facecam. **Update video paths** before using.

### **5. multiview-advanced.yaml**
Professional production with dynamic layouts. **Complete reference example**.

---

## 🎥 **Recording Your Videos**

### **For Split-Screen Tutorials**
1. Record screen with **OBS Studio** or similar
2. Record webcam simultaneously (or separately)
3. Make sure both videos are **same duration** or use audio to align

### **For Gaming PiP**
1. Record gameplay
2. Record facecam
3. Overlay will be automatically positioned

### **For Interviews**
1. Record two separate video sources
2. Or split a single video into two parts

### **For Team Meetings**
1. Record each participant separately, OR
2. Use Zoom/Teams local recording for each person

---

## ⚙️ **Configuration Tips**

### **Video Paths**
```yaml
# Absolute paths
videos:
  left: C:/Users/you/Videos/screen.mp4
  right: C:/Users/you/Videos/webcam.mp4

# Relative paths (from config file location)
videos:
  left: ../videos/screen.mp4
  right: ../videos/webcam.mp4

# Relative to project root
videos:
  left: ./data/videos/screen.mp4
  right: ./data/videos/webcam.mp4
```

### **Slide Ranges**
```yaml
slides: all           # All slides
slides: 0             # Single slide
slides: 0-5           # Range (slides 0 through 5)
slides: [1, 3, 5]     # Specific slides only
```

### **Ratios**
```yaml
ratio: 50:50    # Equal split
ratio: 60:40    # 60% left/top, 40% right/bottom
ratio: 70:30    # 70/30 split
ratio: 65:35    # Common tutorial ratio
```

### **PiP Sizes**
```yaml
size: 20%       # 20% of output width
size: 15%       # Smaller overlay
size: 30%       # Larger overlay
size: 50%       # Half-screen
```

---

## 🔍 **Testing Without Multi-View**

If you want to test without multi-view first:

```yaml
multi_view:
  enabled: false  # Disable multi-view
```

This will generate a normal video from slides + audio, then you can enable multi-view once ready.

---

## 💡 **Common Use Cases**

### **Tutorial Videos**
- Screen recording + instructor webcam
- Use: `split-horizontal` with 65:35 ratio

### **Product Reviews**
- Product close-up + reviewer
- Use: `pip` with bottom-right position

### **Coding Streams**
- Full-screen code + small facecam
- Use: `pip` with 15-20% size

### **Presentations**
- Slides + presenter
- Use: `split-horizontal` with 70:30 ratio

### **Gaming**
- Gameplay + facecam
- Use: `pip` with 15-20% size, custom colors

### **Interviews**
- Two people talking
- Use: `split-horizontal` with 50:50 ratio

### **Webinars**
- Main speaker + participants gallery
- Use: `focus-gallery` layout

---

## 🐛 **Troubleshooting**

### **"File not found" errors**
- Check video file paths are correct
- Use absolute paths if relative paths don't work
- Ensure files exist before running

### **"Slides and audio count mismatch"**
- Multi-view videos should match slide duration
- Use audio length to control segment duration

### **"Invalid layout configuration"**
- Check YAML syntax (proper indentation)
- Ensure all required fields are present
- Validate layout type is supported

---

## 📚 **More Information**

- See `MULTIVIEW_COMPLETE.md` for full feature documentation
- See `MULTI_VIEW_PLAN.md` for implementation details
- See `MULTIVIEW_IMPLEMENTATION.md` for technical reference

---

## 🎉 **Summary**

The example YAMLs are **configuration templates**. To use them:

1. ✅ Prepare your video files
2. ✅ Update paths in YAML
3. ✅ Ensure data/texts.txt and data/slides/ exist
4. ✅ Run: `gocreator create --config your-config.yaml`

**The examples show HOW to configure, you provide the actual videos!**
