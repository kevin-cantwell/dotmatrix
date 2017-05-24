package dotmatrix

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type MJPEGAnimator struct {
	w io.Writer
	d draw.Drawer
	t Terminal
}

func NewMJPEGAnimator(w io.Writer, d draw.Drawer, t Terminal) *MJPEGAnimator {
	if t == nil {
		t = &Xterm{
			Writer: w,
		}
	}
	if d == nil {
		d = draw.FloydSteinberg
	}
	return &MJPEGAnimator{
		w: w,
		d: d,
		t: t,
	}
}

/*
	Animate animates an mpeg stream
*/
func (a *MJPEGAnimator) Animate(r io.Reader, fps int) error {
	a.t.ShowCursor(false)
	defer a.t.ShowCursor(true)
	go a.handleInterrupt()

	enc := NewEncoder(a.w, nil)

	reader := MJPEGReader{Reader: r}
	for frame := range reader.ReadAll() {
		if frame.err != nil {
			return frame.err
		}

		delay := time.After(time.Second / time.Duration(fps))

		// Draw the image and reset the cursor
		if err := enc.Encode(frame.img); err != nil {
			return err
		}
		rows := frame.img.Bounds().Dy() / 4
		if frame.img.Bounds().Dy()%4 != 0 {
			rows++
		}
		a.t.ResetCursor(rows)

		<-delay
	}

	return nil
}

func (a *MJPEGAnimator) handleInterrupt() {
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

/*
	PlayMJPEG is an experimental function that will draw each frame of an MJPEG
	stream to the given writer (usually os.Stdout). Terminal codes are used to
	reposition the cursor at the beginning of each frame.
*/
// func PlayMJPEG(w io.Writer, r io.Reader, fps int) error {
// 	width, height, err := getTerminalSize()
// 	if err != nil {
// 		return err
// 	}
// 	height -= 4

// 	w.Write([]byte("\033[?25l"))                // Hide cursor
// 	defer w.Write([]byte("\033[?12l\033[?25h")) // Show cursor

// 	signals := make(chan os.Signal)
// 	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
// 	go func() {
// 		s := <-signals
// 		w.Write([]byte("\033[?12l\033[?25h")) // Show cursor
// 		// Stop notifying this channel
// 		signal.Stop(signals)
// 		// All Signals returned by the signal package should be of type syscall.Signal
// 		if signum, ok := s.(syscall.Signal); ok {
// 			syscall.Kill(syscall.Getpid(), signum)
// 		} else {
// 			panic(fmt.Sprintf("unexpected signal: %v", s))
// 		}
// 	}()

// 	mjpegs := NewMJPEGScanner(r)
// 	// for range time.Tick(time.Second / time.Duration(fps)) {
// 	// 	if !mjpegs.Scan() {
// 	// 		break
// 	// 	}
// 	ticker := time.Tick(time.Second / time.Duration(fps))
// 	for mjpegs.Scan(ticker) {
// 		img := mjpegs.Image()
// 		img = resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
// 		img = imaging.Invert(img)
// 		img = imaging.FlipH(img)
// 		if err := flush(w, img); err != nil {
// 			return err
// 		}
// 		select {
// 		case <-ticker:
// 		default:
// 		}
// 	}
// 	return mjpegs.Err()
// }

// func flush(w io.Writer, img image.Image) error {
// 	return nil
// }

type frame struct {
	img image.Image
	err error
}

type MJPEGReader struct {
	Reader io.Reader
}

func (mjpeg *MJPEGReader) ReadAll() <-chan frame {
	frames := make(chan frame)
	go func() {
		defer close(frames)

		var buf bytes.Buffer
		p := make([]byte, 1)
		for {
			n, err := mjpeg.Reader.Read(p)
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
				}
			}
		}
	}()
	return frames
}

// type MJPEGScanner struct {
// 	rdr io.Reader
// 	img image.Image
// 	err error
// }

// func NewMJPEGScanner(r io.Reader) *MJPEGScanner {
// 	return &MJPEGScanner{
// 		rdr: r,
// 	}
// }

// func (s *MJPEGScanner) Scan(ticker <-chan time.Time) bool {
// 	p := make([]byte, 1)
// 	var buf bytes.Buffer
// 	for {
// 		n, err := s.rdr.Read(p)
// 		if n == 0 {
// 			if err == nil {
// 				continue
// 			}
// 			if err != io.EOF {
// 				s.err = err
// 			}
// 			return false
// 		}

// 		if _, err := buf.Write(p); err != nil {
// 			s.err = err
// 			return false
// 		}

// 		if buf.Len() > 1 {
// 			data := buf.Bytes()
// 			if data[buf.Len()-2] == 0xff && data[buf.Len()-1] == 0xd9 {
// 				select {
// 				case <-ticker:
// 					img, err := jpeg.Decode(&buf)
// 					if err != nil {
// 						s.err = err
// 						return false
// 					}
// 					s.img = img
// 					return true
// 				default:
// 					buf.Truncate(0)
// 				}
// 			}
// 		}
// 	}
// }

// func (s *MJPEGScanner) Err() error {
// 	return s.err
// }

// func (s *MJPEGScanner) Image() image.Image {
// 	return s.img
// }
