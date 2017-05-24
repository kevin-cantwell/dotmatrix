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

type GIFAnimator struct {
	e *Encoder
	t Terminal
}

func NewGIFAnimator(w io.Writer, f Filter, t Terminal) *GIFAnimator {
	if f == nil {
		f = diffuseFilter{}
	}
	if t == nil {
		t = &Xterm{
			Writer: w,
		}
	}
	return &GIFAnimator{
		e: NewEncoder(w, f),
		t: t,
	}
}

/*
	Animate animates a gif
*/
func (a *GIFAnimator) Animate(giff *gif.GIF) error {
	if len(giff.Image) < 1 {
		return nil
	}

	a.t.ShowCursor(false)
	defer a.t.ShowCursor(true)
	go a.handleInterrupt()

	// Only used if we see background disposal methods
	bgPallette := []color.Color{color.Transparent}
	if giff.Config.ColorModel != nil {
		bgPallette = giff.Config.ColorModel.(color.Palette)
	}

	// The screen is what we flush to the writer on each iteration
	var screen *image.Paletted

	for c := 0; giff.LoopCount == 0 || c < giff.LoopCount; c++ {
		for i := 0; i < len(giff.Image); i++ {
			delay := time.After(time.Duration(giff.Delay[i]) * time.Second / 100)
			frame := convertToMonochrome(a.e.f, giff.Image[i])

			// Always draw the first frame from scratch
			if i == 0 {
				screen = convertToMonochrome(a.e.f, image.NewPaletted(frame.Bounds(), bgPallette))
			}

			switch giff.Disposal[i] {

			// Dispose previous essentially means draw then undo
			case gif.DisposalPrevious:
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := a.flush(screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose background replaces everything just drawn with the background canvas
			case gif.DisposalBackground:
				background := convertToMonochrome(a.e.f, image.NewPaletted(frame.Bounds(), bgPallette))
				drawExact(screen, background)
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := a.flush(screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose none or undefined means we just draw what we got over top
			default:
				drawOver(screen, frame)
				if err := a.flush(screen); err != nil {
					return err
				}
				<-delay
			}
		}
	}
	return nil
}

func (a *GIFAnimator) flush(img image.Image) error {
	if err := a.e.Encode(img); err != nil {
		return err
	}

	rows := img.Bounds().Dy() / 4
	if img.Bounds().Dy()%4 != 0 {
		rows++
	}
	a.t.ResetCursor(rows)

	return nil
}

func (a *GIFAnimator) handleInterrupt() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		a.t.ShowCursor(true)
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
func drawOver(target *image.Paletted, source image.Image) {
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
func drawExact(target *image.Paletted, source image.Image) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			target.Set(x, y, source.At(x, y))
		}
	}
}
