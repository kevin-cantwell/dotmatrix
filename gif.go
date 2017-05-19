package dotmatrix

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"time"
)

var (
	XtermFrameDelim = func(w io.Writer, frame image.Image) {
		w.Write([]byte("\033[999D"))                                    // Move the cursor to the beginning of the line
		w.Write([]byte(fmt.Sprintf("\033[%dA", frame.Bounds().Dy()/4))) // Move the cursor to the top of the image
	}
)

type Animator struct {
	w          io.Writer
	frameDelim func(w io.Writer, frame image.Image)
}

func (a *Animator) Animate(frames <-chan image.Image) error {
	enc := NewEncoder(a.w, nil)
	for frame := range frames {
		if err := enc.Encode(frame); err != nil {
			return err
		}
	}
	return nil
}

type GIFOptions struct {
	Drawer    draw.Drawer
	PreFrame  func(w io.Writer, frame image.Image)
	PostFrame func(w io.Writer, frame image.Image)
}

type GIFEncoder struct {
	w io.Writer
	o GIFOptions
}

func NewGIFEncoder(w io.Writer, opts *GIFOptions) *GIFEncoder {
	o := GIFOptions{}
	if opts != nil {
		o = *opts
	}
	if o.Drawer == nil {
		o.Drawer = draw.FloydSteinberg
	}
	if o.PreFrame == nil {
		o.PreFrame = func(io.Writer, image.Image) {}
	}
	if o.PostFrame == nil {
		o.PostFrame = func(w io.Writer, frame image.Image) {
			height := frame.Bounds().Dy() / 4
			if frame.Bounds().Dy()%4 != 0 {
				height++
			}
			w.Write([]byte("\033[999D"))                     // Move the cursor to the beginning of the line
			w.Write([]byte(fmt.Sprintf("\033[%dA", height))) // Move the cursor to the top of the image
		}
	}
	return &GIFEncoder{
		w: w,
		o: o,
	}
}

/*
	Encode is an experimental function that will draw each frame of a gif to
	the encoder's writer (usually os.Stdout).
*/
func (enc *GIFEncoder) Encode(giff *gif.GIF) error {
	if len(giff.Image) < 1 {
		return nil
	}

	// The screen is what we flush to the writer on each iteration
	var screen *image.Paletted
	// Only used if we see background disposal methods
	bgPallette := color.Palette{color.Transparent}
	if p, ok := giff.Config.ColorModel.(color.Palette); ok {
		bgPallette = p
	}

	for c := 0; giff.LoopCount == 0 || c < giff.LoopCount; c++ {
		for i := 0; i < len(giff.Image); i++ {
			delay := time.After(time.Duration(giff.Delay[i]) * time.Second / 100)
			frame := convert(enc.o.Drawer, giff.Image[i])

			// Always draw the first frame from scratch
			if i == 0 {
				screen = convert(enc.o.Drawer, image.NewPaletted(frame.Bounds(), bgPallette))
			}

			switch giff.Disposal[i] {

			// Dispose previous essentially means draw then undo
			case gif.DisposalPrevious:
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := enc.flush(screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose background replaces everything just drawn with the background canvas
			case gif.DisposalBackground:
				background := convert(enc.o.Drawer, image.NewPaletted(frame.Bounds(), bgPallette))
				drawExact(screen, background)
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := enc.flush(screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose none or undefined means we just draw what we got over top
			default:
				drawOver(screen, frame)
				if err := enc.flush(screen); err != nil {
					return err
				}
				<-delay
			}
		}
	}
	return nil
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

func (enc *GIFEncoder) flush(img image.Image) error {
	enc.o.PreFrame(enc.w, img)
	defer enc.o.PostFrame(enc.w, img)

	if err := NewEncoder(enc.w, enc.o.Drawer).Encode(img); err != nil {
		return err
	}

	return nil
}
