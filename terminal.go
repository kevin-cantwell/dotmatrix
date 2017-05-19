package dotmatrix

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"

	"golang.org/x/crypto/ssh/terminal"
)

// dotmatrix.Screen{}.DrawImage(image.Image)
// dotmatrix.Screen{}.PlayGIF(*image.GIF)

type Terminal struct {
	// Gamma less than 0 darkens the image and GAMMA greater than 0 lightens it.
	Gamma float64
	// Brightness = -100 gives solid black image. Brightness = 100 gives solid white image.
	Brightness float64
	// Contrast = -100 gives solid grey image. Contrast = 100 gives maximum contrast.
	Contrast float64
	// Sharpen greater than 0 sharpens the image.
	Sharpen float64
	// Inverts pixel color. Transparent pixels remain transparent.
	Invert bool
	// MaxWidth is the maximum width of the terminal in number of columns.
	// If zero, a default value will be calculated.
	MaxWidth int
	// MaxHeight is the maximum height of the terminal in number of rows.
	// If zero, a default value will be applied.
	MaxHeight int
	// Drawer specifies the algorithm for drawing images. If nil, draw.FloydSteinberg is used.
	Drawer draw.Drawer
}

func (t *Terminal) DrawImage(img image.Image) error {
	tw, th := t.size()
	scalar := t.scalar(img.Bounds().Dx(), img.Bounds().Dy(), tw, th)
	img = t.resize(img, scalar)
	img = t.correct(img)

	return NewEncoder(t.writer(), t.drawer()).Encode(img)
}

func (t *Terminal) PlayGIF(giff *gif.GIF) error {
	if len(giff.Image) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.handleInterrupt(ctx, func() {
		t.showCursor()
		t.setDefaultTextColor()
	})

	t.hideCursor()

	tw, th := t.size()
	scalar := t.scalar(giff.Config.Width, giff.Config.Height, tw, th)

	newGiff := &gif.GIF{
		Config: image.Config{
			Width:  int(float64(giff.Config.Width) * scalar),
			Height: int(float64(giff.Config.Height) * scalar),
		},
		Delay:           giff.Delay,
		LoopCount:       giff.LoopCount,
		Disposal:        giff.Disposal,
		BackgroundIndex: giff.BackgroundIndex,
	}

	for _, frame := range giff.Image {
		img := t.resize(frame, scalar)
		img = t.correct(img)

		// Create a new paletted image using a monochrome+transparent color palette.
		// If an image adjustment has occurred, we must redefine the bounds so that
		// we maintain the starting point. Not all images start at (0,0) after all.
		paletted := image.NewPaletted(img.Bounds(), simplePalette)
		minX := frame.Bounds().Min.X // Important to use the original frame mins here!
		minY := frame.Bounds().Min.Y
		offset := image.Pt(int(float64(minX)*scalar), int(float64(minY)*scalar))
		paletted.Rect = paletted.Bounds().Add(offset)

		t.drawer().Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)

		newGiff.Image = append(newGiff.Image, paletted)
	}

	return NewGIFEncoder(t.writer(), &GIFOptions{Drawer: t.drawer()}).Encode(newGiff)
}

func (t *Terminal) correct(img image.Image) image.Image {
	if t.Gamma != 0 {
		img = imaging.AdjustGamma(img, t.Gamma+1.0)
	}
	if t.Brightness != 0 {
		img = imaging.AdjustBrightness(img, t.Brightness)
	}
	if t.Sharpen != 0 {
		img = imaging.Sharpen(img, t.Sharpen)
	}
	if t.Contrast != 0 {
		img = imaging.AdjustContrast(img, t.Contrast)
	}
	if t.Invert {
		img = imaging.Invert(img)
	}
	return img
}

func (t *Terminal) resize(img image.Image, scalar float64) image.Image {
	if scalar >= 1.0 {
		return img
	}
	dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
	width := uint(scalar * float64(dx))
	height := uint(scalar * float64(dy))
	return resize.Resize(width, height, img, resize.NearestNeighbor)
}

func (t *Terminal) scalar(dx, dy int, tw, th int) float64 {
	scale := float64(1.0)
	scaleX := float64(tw*2) / float64(dx)
	scaleY := float64(th*4) / float64(dy)

	if scaleX < scale {
		scale = scaleX
	}
	if scaleY < scale {
		scale = scaleY
	}

	return scale
}

func (t *Terminal) size() (int, int) {
	width, height := t.MaxWidth, t.MaxHeight
	if width != 0 && height != 0 {
		return width, height
	}
	if file, ok := t.writer().(*os.File); ok {
		if terminal.IsTerminal(int(file.Fd())) {
			tw, th, err := terminal.GetSize(int(file.Fd()))
			if err == nil {
				th -= 1 // Accounts for the terminal prompt
				if width == 0 {
					width = tw
				}
				if height == 0 {
					height = th
				}
			}
		}
	}
	// Small, but fairly standard defaults
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 25
	}
	return width, height
}

func (t *Terminal) handleInterrupt(ctx context.Context, callback func()) {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		select {
		case s := <-signals:
			callback()
			// Stop notifying this channel
			signal.Stop(signals)
			// All Signals returned by the signal package should be of type syscall.Signal
			if signum, ok := s.(syscall.Signal); ok {
				// Calling os.Exit here would be a bad idea if there are other goroutines
				// waiting to catch signals.
				syscall.Kill(syscall.Getpid(), signum)
			} else {
				panic(fmt.Sprintf("unexpected signal: %v", s))
			}
		case <-ctx.Done():
			callback()
		}
	}()
}

func (t *Terminal) zeroCursorPosition() {
	t.writer().Write([]byte("\033[999D"))                          // Move the cursor to the beginning of the line
	t.writer().Write([]byte(fmt.Sprintf("\033[%dA", t.MaxHeight))) // Move the cursor to the top of the image
}

func (t *Terminal) setDefaultTextColor() {
	t.writer().Write([]byte("\033[0m"))
}

func (t *Terminal) hideCursor() {
	t.writer().Write([]byte("\033[?25l"))
}

func (t *Terminal) showCursor() {
	t.writer().Write([]byte("\033[?12l\033[?25h"))
}

func (t *Terminal) writer() io.Writer {
	return os.Stdout
}

func (t *Terminal) drawer() draw.Drawer {
	if t.Drawer != nil {
		return t.Drawer
	}
	return draw.FloydSteinberg
}
