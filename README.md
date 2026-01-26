# Dotmatrix

[![Go Reference](https://pkg.go.dev/badge/github.com/kevin-cantwell/dotmatrix.svg)](https://pkg.go.dev/github.com/kevin-cantwell/dotmatrix)

Convert images to Unicode braille art for terminal display.

<table>
<tr>
<td><strong>Input</strong></td>
<td><strong>Output</strong></td>
</tr>
<tr>
<td><img src="https://cloud.githubusercontent.com/assets/307864/14945003/a928affe-0fd3-11e6-9725-ae6824be4317.png" alt="Input image" width="300"/></td>
<td><img src="https://cloud.githubusercontent.com/assets/307864/14945005/c9b0d53a-0fd3-11e6-9b06-841eb637a2a0.png" alt="Terminal output" width="300"/></td>
</tr>
</table>

## Features

- Encode JPEG, PNG, GIF, and BMP images as braille Unicode characters
- Animated GIF support with proper frame timing and disposal methods
- MP4 video playback with H.264 decoding (requires FFmpeg)
- Embedded subtitle rendering for MP4 files (mov_text format)
- Native webcam capture on macOS (AVFoundation)
- Image adjustments: gamma, brightness, contrast, sharpening
- Floyd-Steinberg dithering for grayscale preservation
- Automatic scaling to fit terminal dimensions

## Installation

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/kevin-cantwell/dotmatrix/releases). Binaries are available for:
- Linux (amd64)
- macOS (arm64, amd64)

**macOS users:** The binaries are not signed with an Apple Developer account. After downloading, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine dotmatrix
```

### Install with Go

```bash
go install github.com/kevin-cantwell/dotmatrix/cmd/dotmatrix@latest
```

### Building with MP4 Support

MP4 playback requires FFmpeg development libraries and CGO.

**macOS:**
```bash
brew install ffmpeg
PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig" CGO_ENABLED=1 go build -o dotmatrix ./cmd/dotmatrix
codesign -s - dotmatrix  # Ad-hoc sign to prevent "Killed: 9" errors
```

**Ubuntu/Debian:**
```bash
sudo apt-get install libavcodec-dev libavformat-dev libavutil-dev libswscale-dev
CGO_ENABLED=1 go install github.com/kevin-cantwell/dotmatrix/cmd/dotmatrix@latest
```

## Usage

### Command Line

```bash
# From file
dotmatrix image.png

# From URL
dotmatrix https://example.com/image.jpg

# From stdin
curl -s https://example.com/image.jpg | dotmatrix

# With options
dotmatrix --invert --sharpen 50 image.png

# Play MP4 video (subtitles are displayed if embedded)
dotmatrix video.mp4

# Play MP4 at specific framerate
dotmatrix --fps 15 video.mp4

# Capture from webcam (macOS only)
dotmatrix --webcam

# Webcam with options
dotmatrix --webcam --invert --fps 15
```

### As a Library

```go
package main

import (
    "image"
    "os"

    "github.com/kevin-cantwell/dotmatrix"
)

func main() {
    img, _, _ := image.Decode(os.Stdin)
    dotmatrix.Print(os.Stdout, img)
}
```

## Options

| Flag | Description |
|------|-------------|
| `--invert`, `-i` | Invert colors (for dark backgrounds) |
| `--gamma`, `-g` | Adjust gamma: negative darkens, positive lightens |
| `--brightness`, `-b` | Adjust brightness (-100 to 100) |
| `--contrast`, `-c` | Adjust contrast (-100 to 100) |
| `--sharpen`, `-s` | Sharpen image |
| `--mirror`, `-m` | Flip image horizontally |
| `--mono` | Disable Floyd-Steinberg dithering |
| `--webcam`, `-w` | Capture from webcam (macOS only) |
| `--framerate`, `--fps` | Set playback framerate |
| `--mimeType`, `--mime` | Override auto-detected MIME type |

## Examples

### Sharpened Image

```bash
dotmatrix --sharpen 100 face.jpg
```

```
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⣿⣜⢽⡺⣿⣿⣺⣿⣏⣿⣿⣿⣿⢿⣟⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣾⣿⣿⣿⣿⣿⣿⣿⣿⣽⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣜⡞⣧⢅⢳⡙⣼⣻⡢⡺⡼⣻⢞⡯⣟⣽⢻⡿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣻⣯⣿⣿⣿⡿⡿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡜⣖⢔⠕⢸⡳⡢⡱⠅⡞⣽⠱⢳⢫⡟⡞⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣱⣿⣿⣵⡿⣿⣟⡿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣽⣷⢿⡽⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⡿⡽⣿⢿⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣮⢣⢫⠀⠣⠂⢘⠐⢁⢊⠠⠉⢎⢎⢣⢿⡻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡏⡼⡜⡾⡯⡫⢽⢽⣗⢟⣿⣿⣿⣿⣿⣿⣿⣟⡮⣟⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⡿⡭⡺⣫⣟⡷⣟⢾⢽⢿⣟⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣎⠄⢊⢂⠠⠈⠄⠂⠀⢁⢑⠁⢜⡧⡗⢡⢮⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣜⢜⢌⢊⠈⣄⢜⢿⡫⣿⣿⣿⣿⣿⣿⣿⡿⡱⣻⢟⣿⣽⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
```

### Animated GIF

Animated GIFs play directly in the terminal with proper timing:

![Animated GIF example](https://cloud.githubusercontent.com/assets/307864/16272242/0dc3b6d8-386b-11e6-9ea3-e55ee936ae54.gif)

> **Note:** Terminal refresh rates vary. The default macOS Terminal.app works well; iTerm2 may have slower refresh rates.

### MP4 Subtitles

MP4 files with embedded subtitles (mov_text/tx3g format) will display subtitles overlaid at the bottom of the video. Subtitles are:

- Automatically extracted and timed to video playback
- Centered and wrapped to fit the terminal width
- Rendered over the braille output while preserving surrounding pixels

## How It Works

Dotmatrix uses [Unicode Braille Patterns](https://en.wikipedia.org/wiki/Braille_Patterns) (U+2800 to U+28FF) to represent images. Each braille character encodes a 2x4 pixel grid, allowing 256 possible patterns per character.

**Processing pipeline:**

1. **Decode** - Parse input (JPEG, PNG, GIF, BMP, or MP4 video)
2. **Filter** - Apply brightness, contrast, gamma, sharpening adjustments
3. **Scale** - Resize to fit terminal dimensions (2 pixels per column, 4 pixels per row)
4. **Dither** - Convert to monochrome using Floyd-Steinberg diffusion
5. **Encode** - Map each 2x4 pixel block to a braille character
6. **Render** - Output braille characters with newlines (for video, frames are rendered in sequence)

The Floyd-Steinberg dithering algorithm distributes quantization errors to neighboring pixels, preserving the appearance of grayscale gradients in the monochrome output.

## License

MIT
