package dotmatrix

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"os/signal"
	"syscall"
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
func (a *MJPEGPrinter) Print(r io.Reader, fps int) error {
	showCursor(a.w, false)
	defer showCursor(a.w, true)
	go a.handleInterrupt()

	reader := mjpegStreamer{
		r:   r,
		fps: fps,
	}
	for frame := range reader.ReadAll() {
		if frame.err != nil {
			return frame.err
		}

		frame.img = redraw(frame.img, a.c.Filter, a.c.Drawer)

		// Draw the image and reset the cursor
		if err := flushBraille(a.w, frame.img); err != nil {
			return err
		}
		rows := frame.img.Bounds().Dy() / 4
		if frame.img.Bounds().Dy()%4 != 0 {
			rows++
		}

		resetCursor(a.w, rows)
	}

	return nil
}

func (a *MJPEGPrinter) handleInterrupt() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		showCursor(a.w, true)
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

type frame struct {
	img image.Image
	err error
}

type mjpegStreamer struct {
	r   io.Reader
	fps int
}

func (mjpeg *mjpegStreamer) ReadAll() <-chan frame {
	frames := make(chan frame)
	go func() {
		defer close(frames)

		var buf bytes.Buffer
		p := make([]byte, 1)
		for {
			delay := time.After(time.Second / time.Duration(mjpeg.fps))

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
					case frames <- frame{img: img, err: err}:
					default:
						buf.Truncate(0)
					}
					<-delay
				}
			}
		}
	}()
	return frames
}
