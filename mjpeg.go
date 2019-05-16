package dotmatrix

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"time"
)

type MJPEGPrinter struct {
	w io.Writer
	c Config
}

func NewMJPEGPrinter(w io.Writer, c *Config) *MJPEGPrinter {
	return &MJPEGPrinter{
		w: w,
		c: mergeConfig(c),
	}
}

/*
	Print animates an mpeg stream. If fps is less than zero, it will print each
	frame as quickly as it can. Otherwise, fps dictacts how many frames per second
	are printed.
*/
func (p *MJPEGPrinter) Print(ctx context.Context, r io.Reader, fps int) error {
	reader := mjpegStreamer{
		r:   r,
		fps: fps,
	}

	for frame := range reader.ReadAll(ctx) {
		if frame.err != nil {
			return frame.err
		}

		frame.img = redraw(frame.img, p.c.Filter, p.c.Drawer)

		// Draw the image and reset the cursor
		if err := flush(p.w, frame.img, p.c.Flusher); err != nil {
			return err
		}
		rows := frame.img.Bounds().Dy() / 4
		if frame.img.Bounds().Dy()%4 != 0 {
			rows++
		}

		p.c.Reset(p.w, rows)
	}

	return nil
}

type frame struct {
	img image.Image
	err error
}

type mjpegStreamer struct {
	r   io.Reader
	fps int
}

func (mjpeg *mjpegStreamer) ReadAll(ctx context.Context) <-chan frame {
	frames := make(chan frame)
	go func() {
		defer close(frames)

		var buf bytes.Buffer
		p := make([]byte, 1)
		delay := time.After(time.Second / time.Duration(mjpeg.fps))
		for {
			n, err := mjpeg.r.Read(p)
			if n == 0 {
				if err == nil {
					continue
				}
				if err != io.EOF {
					frames <- frame{err: err}
				}
				return
			}

			if _, err := buf.Write(p); err != nil {
				frames <- frame{err: err}
				return
			}

			if buf.Len() > 1 {
				data := buf.Bytes()
				if data[buf.Len()-2] == 0xff && data[buf.Len()-1] == 0xd9 {
					img, err := jpeg.Decode(&buf)
					if err != nil {
						frames <- frame{err: err}
						return
					}
					select {
					case <-ctx.Done():
						return
					case frames <- frame{img: img, err: err}:
						<-delay
					default:
						buf.Truncate(0)
					}
					delay = time.After(time.Second / time.Duration(mjpeg.fps))
				}
			}
		}
	}()
	return frames
}
