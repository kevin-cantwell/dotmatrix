package dotmatrix

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type GIFPrinter struct {
	w io.Writer
	c Config
}

func NewGIFPrinter(w io.Writer, c *Config) *GIFPrinter {
	return &GIFPrinter{
		w: w,
		c: mergeConfig(c),
	}
}

/*
	Print animates a gif
*/
func (p *GIFPrinter) Print(giff *gif.GIF) error {
	if len(giff.Image) < 1 {
		return nil
	}

	showCursor(p.w, false)
	defer showCursor(p.w, true)
	go p.handleInterrupt()

	// Only used if we see background disposal methods
	bgPallette := []color.Color{color.Transparent}
	if giff.Config.ColorModel != nil {
		bgPallette = giff.Config.ColorModel.(color.Palette)
	}

	// The screen is what we flush to the writer on each iteration
	screen := redraw(image.NewPaletted(giff.Image[0].Bounds(), bgPallette), p.c.Filter, p.c.Drawer)
	rows := giff.Config.Height / 4
	if giff.Config.Height%4 != 0 {
		rows++
	}

	for c := 0; giff.LoopCount == 0 || c < giff.LoopCount; c++ {
		for i := 0; i < len(giff.Image); i++ {
			delay := time.After(time.Duration(giff.Delay[i]) * time.Second / 100)

			frame := redraw(giff.Image[i], p.c.Filter, p.c.Drawer)

			switch giff.Disposal[i] {
			case gif.DisposalPrevious: // Dispose previous essentially means draw then undo
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				p.drawOver(screen, frame)
				if err := flushBraille(p.w, screen); err != nil {
					return err
				}
				<-delay

				screen = temp
			case gif.DisposalBackground: // Dispose background replaces everything just drawn with the background canvas
				background := redraw(image.NewPaletted(frame.Bounds(), bgPallette), p.c.Filter, p.c.Drawer)
				p.drawExact(screen, background)
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				p.drawOver(screen, frame)
				if err := flushBraille(p.w, screen); err != nil {
					return err
				}
				<-delay

				screen = temp
			default: // Dispose none or undefined means we just draw what we got over top
				p.drawOver(screen, frame)
				if err := flushBraille(p.w, screen); err != nil {
					return err
				}
				<-delay
			}

			resetCursor(p.w, rows)
		}
	}
	return nil
}

func (p *GIFPrinter) handleInterrupt() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		showCursor(p.w, true)
		// Stop notifying this channel
		signal.Stop(signals)
		// All Signals returned by the signal package should be of type syscall.Signal
		if signum, ok := s.(syscall.Signal); ok {
			// Calling os.Exit here would be a bad idea if there are other goroutines
			// waiting to catch the same signal.
			syscall.Kill(syscall.Getpid(), signum)
		} else {
			panic(fmt.Sprintf("unexpected signal: %v", s))
		}
	}()
}

// Draws any non-transparent pixels into target
func (p *GIFPrinter) drawOver(target *image.Paletted, source image.Image) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := source.At(x, y)
			if c == color.Transparent {
				continue
			}
			target.Set(x, y, c)
		}
	}
}

// Draws pixels into target, including transparent ones.
func (p *GIFPrinter) drawExact(target *image.Paletted, source image.Image) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			target.Set(x, y, source.At(x, y))
		}
	}
}

// Move the cursor to the beginning of the line and up rows
func resetCursor(w io.Writer, rows int) {
	w.Write([]byte(fmt.Sprintf("\033[999D\033[%dA", rows)))
}

func showCursor(w io.Writer, show bool) {
	if show {
		w.Write([]byte("\033[?12l\033[?25h"))
	} else {
		w.Write([]byte("\033[?25l"))
	}
}
