package dotmatrix

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"time"
)

/*
PlayGIF is an experimental function that will draw each frame of a gif to
the given writer (usually os.Stdout). Terminal codes are used to reposition
the cursor at the beginning of each frame. Delays and disposal methods are
respected.
*/
func PlayGIF(w io.Writer, giff *gif.GIF) error {
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
			frame := convert(giff.Image[i])

			// Always draw the first frame from scratch
			if i == 0 {
				screen = convert(image.NewPaletted(frame.Bounds(), bgPallette))
			}

			switch giff.Disposal[i] {

			// Dispose previous essentially means draw then undo
			case gif.DisposalPrevious:
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := flush(w, screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose background replaces everything just drawn with the background canvas
			case gif.DisposalBackground:
				background := convert(image.NewPaletted(frame.Bounds(), bgPallette))
				drawExact(screen, background)
				temp := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(temp.Pix, screen.Pix)

				drawOver(screen, frame)
				if err := flush(w, screen); err != nil {
					return err
				}
				<-delay

				screen = temp

			// Dispose none or undefined means we just draw what we got over top
			default:
				drawOver(screen, frame)
				if err := flush(w, screen); err != nil {
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

func flush(w io.Writer, img image.Image) error {
	w.Write([]byte("\033[0m")) // This can be used to hijack writer and detect when we start a new frame
	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		return err
	}
	var height int
	for {
		c, err := buf.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if c == '\n' {
			height++
		}
		w.Write([]byte{c})
	}
	w.Write([]byte("\033[999D"))                     // Move the cursor to the beginning of the line
	w.Write([]byte(fmt.Sprintf("\033[%dA", height))) // Move the cursor to the top of the image
	return nil
}
