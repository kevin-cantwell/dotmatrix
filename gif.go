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
	var screen *image.Paletted

	for c := 0; giff.LoopCount == 0 || c < giff.LoopCount; c++ {
		for i := 0; i < len(giff.Image); i++ {
			delay := time.After(time.Duration(giff.Delay[i]) * time.Second / 100)
			frame := convert(giff.Image[i])

			// Always draw the first frame from scratch
			if i == 0 {
				screen = frame
			}

			switch giff.Disposal[i] {
			// Dispose previous essentially means draw then undo
			case gif.DisposalPrevious:
				previous := image.NewPaletted(screen.Bounds(), screen.Palette)
				copy(previous.Pix, screen.Pix)
				drawFrame(screen, frame)
				if err := flush(w, screen); err != nil {
					return err
				}
				screen = previous
				<-delay
			// Dispose background replaces everything just drawn with the background canvas
			case gif.DisposalBackground:
				background := image.NewPaletted(frame.Bounds(), frame.Palette)
				for i := 0; i < len(background.Pix); i++ {
					background.Pix[i] = 2
				}
				drawFrame(screen, frame)
				if err := flush(w, screen); err != nil {
					return err
				}
				<-delay
				drawFrame(screen, background)
				if err := flush(w, screen); err != nil {
					return err
				}
			// Dispose none or undefined means we just draw what we got over top
			default:
				drawFrame(screen, frame)
				if err := flush(w, screen); err != nil {
					return err
				}
				<-delay
			}
		}
	}
	return nil
}

func drawFrame(target *image.Paletted, source image.Image) {
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

func flush(w io.Writer, img image.Image) error {
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
