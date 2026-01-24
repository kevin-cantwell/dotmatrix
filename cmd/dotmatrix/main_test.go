package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevin-cantwell/dotmatrix"
	"github.com/urfave/cli/v2"
)

// createTestImage creates a simple test image with the given dimensions
func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

// createTestPNG creates a PNG file and returns its path
// Uses larger dimensions to ensure file is > 512 bytes for mime detection
func createTestPNG(t *testing.T, width, height int) string {
	t.Helper()
	// Ensure minimum size for mime detection (needs at least 512 bytes)
	// PNG compresses simple patterns very well, so we need larger images
	if width < 200 {
		width = 200
	}
	if height < 200 {
		height = 200
	}

	// Create image with gradient (harder to compress than checkerboard)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create gradient pattern that doesn't compress as well
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	return tmpFile.Name()
}

// createTestGIF creates a GIF file and returns its path
// Uses larger dimensions to ensure file is > 512 bytes for mime detection
func createTestGIF(t *testing.T, width, height, frames int) string {
	t.Helper()
	// Ensure minimum size for mime detection (needs at least 512 bytes)
	if width < 100 {
		width = 100
	}
	if height < 100 {
		height = 100
	}

	g := &gif.GIF{
		Image:     make([]*image.Paletted, frames),
		Delay:     make([]int, frames),
		LoopCount: 1,
	}

	palette := []color.Color{color.White, color.Black}

	for i := 0; i < frames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if (x+y+i)%2 == 0 {
					img.SetColorIndex(x, y, 0)
				} else {
					img.SetColorIndex(x, y, 1)
				}
			}
		}
		g.Image[i] = img
		g.Delay[i] = 10
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.gif")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if err := gif.EncodeAll(tmpFile, g); err != nil {
		t.Fatalf("failed to encode GIF: %v", err)
	}

	return tmpFile.Name()
}

func TestScalar(t *testing.T) {
	tests := []struct {
		name     string
		dx, dy   int
		cols     int
		rows     int
		expected float64
	}{
		{
			name:     "image fits without scaling",
			dx:       100,
			dy:       100,
			cols:     80,
			rows:     25,
			expected: 1.0,
		},
		{
			name:     "image needs horizontal scaling",
			dx:       200,
			dy:       50,
			cols:     80,
			rows:     25,
			expected: 0.8, // cols*2/dx = 160/200 = 0.8
		},
		{
			name:     "image needs vertical scaling",
			dx:       50,
			dy:       200,
			cols:     80,
			rows:     25,
			expected: 0.5, // rows*4/dy = 100/200 = 0.5
		},
		{
			name:     "small image no scaling needed",
			dx:       10,
			dy:       10,
			cols:     80,
			rows:     25,
			expected: 1.0,
		},
		{
			name:     "large image needs scaling both dimensions",
			dx:       1000,
			dy:       1000,
			cols:     80,
			rows:     25,
			expected: 0.1, // min(160/1000, 100/1000) = 0.1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scalar(tt.dx, tt.dy, tt.cols, tt.rows)
			if result != tt.expected {
				t.Errorf("scalar(%d, %d, %d, %d) = %v, want %v",
					tt.dx, tt.dy, tt.cols, tt.rows, result, tt.expected)
			}
		})
	}
}

func TestFilter_Gamma(t *testing.T) {
	img := createTestImage(20, 20)

	f := &Filter{Gamma: 0.5, scale: 1.0}
	result := f.Filter(img)

	if result.Bounds().Dx() != 20 || result.Bounds().Dy() != 20 {
		t.Errorf("expected dimensions 20x20, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_Brightness(t *testing.T) {
	img := createTestImage(20, 20)

	f := &Filter{Brightness: 50, scale: 1.0}
	result := f.Filter(img)

	if result.Bounds().Dx() != 20 || result.Bounds().Dy() != 20 {
		t.Errorf("expected dimensions 20x20, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_Contrast(t *testing.T) {
	img := createTestImage(20, 20)

	f := &Filter{Contrast: 50, scale: 1.0}
	result := f.Filter(img)

	if result.Bounds().Dx() != 20 || result.Bounds().Dy() != 20 {
		t.Errorf("expected dimensions 20x20, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_Sharpen(t *testing.T) {
	img := createTestImage(20, 20)

	f := &Filter{Sharpen: 1.0, scale: 1.0}
	result := f.Filter(img)

	if result.Bounds().Dx() != 20 || result.Bounds().Dy() != 20 {
		t.Errorf("expected dimensions 20x20, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_Mirror(t *testing.T) {
	// Create image with distinct left/right sides
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}

	f := &Filter{Mirror: true, scale: 1.0}
	result := f.Filter(img)

	// After mirroring, left side should be black, right side white
	r, g, b, _ := result.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Error("expected left side to be black after mirroring")
	}

	r, g, b, _ = result.At(19, 0).RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff {
		t.Error("expected right side to be white after mirroring")
	}
}

func TestFilter_Invert(t *testing.T) {
	// Create solid white image
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.White)
		}
	}

	f := &Filter{Invert: true, scale: 1.0}
	result := f.Filter(img)

	// After inversion, should be black
	r, g, b, _ := result.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("expected black after inversion, got R=%d G=%d B=%d", r, g, b)
	}
}

func TestFilter_Resize(t *testing.T) {
	img := createTestImage(100, 100)

	// Set scale to 0.5 to halve the image
	f := &Filter{scale: 0.5}
	result := f.Filter(img)

	if result.Bounds().Dx() != 50 || result.Bounds().Dy() != 50 {
		t.Errorf("expected dimensions 50x50, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_CombinedOperations(t *testing.T) {
	img := createTestImage(40, 40)

	f := &Filter{
		Gamma:      0.5,
		Brightness: 10,
		Contrast:   20,
		Sharpen:    0.5,
		Mirror:     true,
		Invert:     true,
		scale:      0.5,
	}
	result := f.Filter(img)

	// Should be resized to half
	if result.Bounds().Dx() != 20 || result.Bounds().Dy() != 20 {
		t.Errorf("expected dimensions 20x20, got %dx%d",
			result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestFilter_ScaleCalculation(t *testing.T) {
	img := createTestImage(1000, 1000)

	// scale starts at 0, should be calculated based on terminal dimensions
	f := &Filter{}
	result := f.Filter(img)

	// Scale should be calculated and image should be smaller than original
	if result.Bounds().Dx() >= 1000 || result.Bounds().Dy() >= 1000 {
		t.Error("expected image to be scaled down")
	}

	// scale should now be set
	if f.scale == 0 {
		t.Error("expected scale to be calculated")
	}
}

func TestTerminalDimensions(t *testing.T) {
	cols, rows := terminalDimensions()

	// When not running in a terminal, should return defaults
	if cols < 1 || rows < 1 {
		t.Errorf("expected positive dimensions, got cols=%d rows=%d", cols, rows)
	}

	// Default values are 80x25
	if cols == 0 {
		t.Error("cols should not be 0")
	}
	if rows == 0 {
		t.Error("rows should not be 0")
	}
}

func TestCLI_FlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		checkFn  func(*cli.Context) error
	}{
		{
			name: "invert flag",
			args: []string{"dotmatrix", "--invert"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("invert") {
					t.Error("expected invert to be true")
				}
				return nil
			},
		},
		{
			name: "invert short flag",
			args: []string{"dotmatrix", "-i"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("invert") {
					t.Error("expected invert to be true with -i")
				}
				return nil
			},
		},
		{
			name: "gamma flag",
			args: []string{"dotmatrix", "--gamma", "0.5"},
			checkFn: func(c *cli.Context) error {
				if c.Float64("gamma") != 0.5 {
					t.Errorf("expected gamma 0.5, got %f", c.Float64("gamma"))
				}
				return nil
			},
		},
		{
			name: "gamma short flag",
			args: []string{"dotmatrix", "-g", "0.5"},
			checkFn: func(c *cli.Context) error {
				if c.Float64("gamma") != 0.5 {
					t.Errorf("expected gamma 0.5, got %f", c.Float64("gamma"))
				}
				return nil
			},
		},
		{
			name: "brightness flag",
			args: []string{"dotmatrix", "--brightness", "50"},
			checkFn: func(c *cli.Context) error {
				if c.Float64("brightness") != 50 {
					t.Errorf("expected brightness 50, got %f", c.Float64("brightness"))
				}
				return nil
			},
		},
		{
			name: "contrast flag",
			args: []string{"dotmatrix", "--contrast", "-25"},
			checkFn: func(c *cli.Context) error {
				if c.Float64("contrast") != -25 {
					t.Errorf("expected contrast -25, got %f", c.Float64("contrast"))
				}
				return nil
			},
		},
		{
			name: "sharpen flag",
			args: []string{"dotmatrix", "--sharpen", "100"},
			checkFn: func(c *cli.Context) error {
				if c.Float64("sharpen") != 100 {
					t.Errorf("expected sharpen 100, got %f", c.Float64("sharpen"))
				}
				return nil
			},
		},
		{
			name: "mirror flag",
			args: []string{"dotmatrix", "--mirror"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("mirror") {
					t.Error("expected mirror to be true")
				}
				return nil
			},
		},
		{
			name: "mono flag",
			args: []string{"dotmatrix", "--mono"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("mono") {
					t.Error("expected mono to be true")
				}
				return nil
			},
		},
		{
			name: "motion flag",
			args: []string{"dotmatrix", "--motion"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("motion") {
					t.Error("expected motion to be true")
				}
				return nil
			},
		},
		{
			name: "motion mjpeg alias",
			args: []string{"dotmatrix", "--mjpeg"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("motion") {
					t.Error("expected motion to be true with --mjpeg alias")
				}
				return nil
			},
		},
		{
			name: "framerate flag",
			args: []string{"dotmatrix", "--framerate", "30"},
			checkFn: func(c *cli.Context) error {
				if c.Int("framerate") != 30 {
					t.Errorf("expected framerate 30, got %d", c.Int("framerate"))
				}
				return nil
			},
		},
		{
			name: "framerate default",
			args: []string{"dotmatrix"},
			checkFn: func(c *cli.Context) error {
				if c.Int("framerate") != -1 {
					t.Errorf("expected default framerate -1, got %d", c.Int("framerate"))
				}
				return nil
			},
		},
		{
			name: "mimeType flag",
			args: []string{"dotmatrix", "--mimeType", "image/gif"},
			checkFn: func(c *cli.Context) error {
				if c.String("mimeType") != "image/gif" {
					t.Errorf("expected mimeType image/gif, got %s", c.String("mimeType"))
				}
				return nil
			},
		},
		{
			name: "multiple flags",
			args: []string{"dotmatrix", "-i", "-m", "--gamma", "0.5", "--brightness", "10"},
			checkFn: func(c *cli.Context) error {
				if !c.Bool("invert") {
					t.Error("expected invert to be true")
				}
				if !c.Bool("mirror") {
					t.Error("expected mirror to be true")
				}
				if c.Float64("gamma") != 0.5 {
					t.Errorf("expected gamma 0.5, got %f", c.Float64("gamma"))
				}
				if c.Float64("brightness") != 10 {
					t.Errorf("expected brightness 10, got %f", c.Float64("brightness"))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.App{
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}},
					&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}},
					&cli.Float64Flag{Name: "brightness", Aliases: []string{"b"}},
					&cli.Float64Flag{Name: "contrast", Aliases: []string{"c"}},
					&cli.Float64Flag{Name: "sharpen", Aliases: []string{"s"}},
					&cli.BoolFlag{Name: "mirror", Aliases: []string{"m"}},
					&cli.BoolFlag{Name: "mono"},
					&cli.BoolFlag{Name: "motion", Aliases: []string{"mjpeg"}},
					&cli.IntFlag{Name: "framerate", Aliases: []string{"fps"}, Value: -1},
					&cli.StringFlag{Name: "mimeType", Aliases: []string{"mime"}},
				},
				Action: tt.checkFn,
			}

			err := app.Run(tt.args)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}},
			&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}},
			&cli.Float64Flag{Name: "brightness", Aliases: []string{"b"}},
			&cli.Float64Flag{Name: "contrast", Aliases: []string{"c"}},
			&cli.Float64Flag{Name: "sharpen", Aliases: []string{"s"}},
			&cli.BoolFlag{Name: "mirror", Aliases: []string{"m"}},
			&cli.BoolFlag{Name: "mono"},
		},
		Action: func(c *cli.Context) error {
			cfg := config(c)

			if cfg.Filter == nil {
				t.Error("expected Filter to be set")
				return nil
			}

			filter := cfg.Filter.(*Filter)

			if filter.Gamma != 0.5 {
				t.Errorf("expected gamma 0.5, got %f", filter.Gamma)
			}
			if filter.Brightness != 10 {
				t.Errorf("expected brightness 10, got %f", filter.Brightness)
			}
			if filter.Contrast != 20 {
				t.Errorf("expected contrast 20, got %f", filter.Contrast)
			}
			if filter.Sharpen != 5 {
				t.Errorf("expected sharpen 5, got %f", filter.Sharpen)
			}
			if !filter.Invert {
				t.Error("expected invert to be true")
			}
			if !filter.Mirror {
				t.Error("expected mirror to be true")
			}

			return nil
		},
	}

	args := []string{"dotmatrix", "-i", "-m", "-g", "0.5", "-b", "10", "-c", "20", "-s", "5"}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfig_MonoDrawer(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}},
			&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}},
			&cli.Float64Flag{Name: "brightness", Aliases: []string{"b"}},
			&cli.Float64Flag{Name: "contrast", Aliases: []string{"c"}},
			&cli.Float64Flag{Name: "sharpen", Aliases: []string{"s"}},
			&cli.BoolFlag{Name: "mirror", Aliases: []string{"m"}},
			&cli.BoolFlag{Name: "mono"},
		},
		Action: func(c *cli.Context) error {
			cfg := config(c)

			// When mono is true, should use draw.Src
			if cfg.Drawer == nil {
				t.Error("expected Drawer to be set")
			}
			return nil
		},
	}

	args := []string{"dotmatrix", "--mono"}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageAction(t *testing.T) {
	pngPath := createTestPNG(t, 40, 40)

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}},
			&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}},
			&cli.Float64Flag{Name: "brightness", Aliases: []string{"b"}},
			&cli.Float64Flag{Name: "contrast", Aliases: []string{"c"}},
			&cli.Float64Flag{Name: "sharpen", Aliases: []string{"s"}},
			&cli.BoolFlag{Name: "mirror", Aliases: []string{"m"}},
			&cli.BoolFlag{Name: "mono"},
		},
		Action: func(c *cli.Context) error {
			f, err := os.Open(pngPath)
			if err != nil {
				return err
			}
			defer f.Close()

			// Capture output
			var buf bytes.Buffer
			cfg := config(c)
			cfg.Filter = &Filter{scale: 1.0}

			img, _, err := image.Decode(f)
			if err != nil {
				return err
			}

			return dotmatrix.NewPrinter(&buf, cfg).Print(img)
		},
	}

	args := []string{"dotmatrix"}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageAction_WithAllFilters(t *testing.T) {
	pngPath := createTestPNG(t, 40, 40)

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}},
			&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}},
			&cli.Float64Flag{Name: "brightness", Aliases: []string{"b"}},
			&cli.Float64Flag{Name: "contrast", Aliases: []string{"c"}},
			&cli.Float64Flag{Name: "sharpen", Aliases: []string{"s"}},
			&cli.BoolFlag{Name: "mirror", Aliases: []string{"m"}},
			&cli.BoolFlag{Name: "mono"},
		},
		Action: func(c *cli.Context) error {
			f, err := os.Open(pngPath)
			if err != nil {
				return err
			}
			defer f.Close()

			var buf bytes.Buffer
			cfg := config(c)

			img, _, err := image.Decode(f)
			if err != nil {
				return err
			}

			return dotmatrix.NewPrinter(&buf, cfg).Print(img)
		},
	}

	args := []string{"dotmatrix", "-i", "-m", "-g", "0.5", "-b", "10", "-c", "20", "-s", "5", "--mono"}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeReader_File(t *testing.T) {
	pngPath := createTestPNG(t, 40, 40)

	app := &cli.App{
		Action: func(c *cli.Context) error {
			reader, mimeType, err := decodeReader(c)
			if err != nil {
				return err
			}

			if reader == nil {
				t.Error("expected reader to be set")
			}

			if !strings.HasPrefix(mimeType, "image/png") {
				t.Errorf("expected image/png mime type, got %s", mimeType)
			}

			return nil
		},
	}

	args := []string{"dotmatrix", pngPath}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeReader_GIF(t *testing.T) {
	gifPath := createTestGIF(t, 40, 40, 2)

	app := &cli.App{
		Action: func(c *cli.Context) error {
			reader, mimeType, err := decodeReader(c)
			if err != nil {
				return err
			}

			if reader == nil {
				t.Error("expected reader to be set")
			}

			if mimeType != "image/gif" {
				t.Errorf("expected image/gif mime type, got %s", mimeType)
			}

			return nil
		},
	}

	args := []string{"dotmatrix", gifPath}
	if err := app.Run(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeReader_NonexistentFile(t *testing.T) {
	app := &cli.App{
		Action: func(c *cli.Context) error {
			_, _, err := decodeReader(c)
			if err == nil {
				t.Error("expected error for nonexistent file")
			}
			return nil
		},
	}

	args := []string{"dotmatrix", "/nonexistent/path/to/file.png"}
	// Run doesn't return error because Action returns nil
	app.Run(args)
}

func TestBrailleOutput(t *testing.T) {
	// Create a simple 4x8 image (will produce 2x2 braille characters)
	img := image.NewRGBA(image.Rect(0, 0, 4, 8))

	// Fill with black (will produce filled braille dots)
	for y := 0; y < 8; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.Black)
		}
	}

	var buf bytes.Buffer
	cfg := &dotmatrix.Config{
		Filter: &Filter{scale: 1.0},
	}

	err := dotmatrix.NewPrinter(&buf, cfg).Print(img)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected braille output")
	}

	// Output should contain braille characters (U+2800 to U+28FF)
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	if !hasBraille {
		t.Error("expected output to contain braille characters")
	}
}

func TestBrailleOutput_WhiteImage(t *testing.T) {
	// Create white image (should produce empty braille)
	img := image.NewRGBA(image.Rect(0, 0, 4, 8))

	for y := 0; y < 8; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.White)
		}
	}

	var buf bytes.Buffer
	cfg := &dotmatrix.Config{
		Filter: &Filter{scale: 1.0},
	}

	err := dotmatrix.NewPrinter(&buf, cfg).Print(img)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	// Empty braille is U+2800
	if !strings.Contains(output, "\u2800") {
		t.Error("expected empty braille character for white image")
	}
}

func TestCLI_Version(t *testing.T) {
	app := &cli.App{
		Name:    "dotmatrix",
		Version: "0.1.0",
	}

	var buf bytes.Buffer
	app.Writer = &buf

	args := []string{"dotmatrix", "--version"}
	err := app.Run(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "0.1.0") {
		t.Errorf("expected version in output, got: %s", buf.String())
	}
}

func TestCLI_Help(t *testing.T) {
	app := &cli.App{
		Name:  "dotmatrix",
		Usage: "A command-line tool for encoding images as unicode braille symbols.",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "invert", Aliases: []string{"i"}, Usage: "Inverts image color"},
			&cli.Float64Flag{Name: "gamma", Aliases: []string{"g"}, Usage: "Adjust gamma"},
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	args := []string{"dotmatrix", "--help"}
	err := app.Run(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "dotmatrix") {
		t.Error("expected app name in help output")
	}
	if !strings.Contains(output, "--invert") {
		t.Error("expected --invert flag in help output")
	}
	if !strings.Contains(output, "--gamma") {
		t.Error("expected --gamma flag in help output")
	}
}

func TestFileExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	// Test PNG
	pngPath := filepath.Join(tmpDir, "test.png")
	pngImg := createTestImage(20, 20)
	pngFile, _ := os.Create(pngPath)
	png.Encode(pngFile, pngImg)
	pngFile.Close()

	f, err := os.Open(pngPath)
	if err != nil {
		t.Fatalf("failed to open PNG: %v", err)
	}
	defer f.Close()

	_, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode PNG: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png format, got %s", format)
	}
}

func TestPrintToWriter(t *testing.T) {
	img := createTestImage(20, 20)

	// Test that we can print to any io.Writer
	var buf bytes.Buffer
	cfg := &dotmatrix.Config{
		Filter: &Filter{scale: 1.0},
	}

	err := dotmatrix.NewPrinter(&buf, cfg).Print(img)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output in buffer")
	}
}

func TestPrintToDiscardWriter(t *testing.T) {
	img := createTestImage(20, 20)

	cfg := &dotmatrix.Config{
		Filter: &Filter{scale: 1.0},
	}

	// Should work with io.Discard
	err := dotmatrix.NewPrinter(io.Discard, cfg).Print(img)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
