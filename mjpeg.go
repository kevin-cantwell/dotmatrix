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

	"golang.org/x/crypto/ssh/terminal"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
)

func getTerminalSize() (int, int, error) {
	w, h, err := terminal.GetSize(int(os.Stdout.Fd()))
	return w * 2, h * 4, err
	// var dimensions [4]uint16
	// _, _, e := syscall.Syscall6(
	// 	syscall.SYS_IOCTL,
	// 	uintptr(syscall.Stderr), // TODO: Figure out why we get "inappropriate ioctl for device" errors if we use stdin or stdout
	// 	uintptr(syscall.TIOCGWINSZ),
	// 	uintptr(unsafe.Pointer(&dimensions)),
	// 	0, 0, 0,
	// )
	// if e != 0 {
	// 	return 160, 0, e
	// }
	// // return uint(dimensions[1]) * 2, uint(dimensions[0]) * 4, nil
	// return uint(dimensions[1]) * 2, uint(dimensions[0]) * 4, nil
}

/*
	PlayMJPEG is an experimental function that will draw each frame of an MJPEG
	stream to the given writer (usually os.Stdout). Terminal codes are used to
	reposition the cursor at the beginning of each frame.
*/
func PlayMJPEG(w io.Writer, r io.Reader, fps int) error {
	width, height, err := getTerminalSize()
	if err != nil {
		return err
	}
	height -= 4

	w.Write([]byte("\033[?25l"))                // Hide cursor
	defer w.Write([]byte("\033[?12l\033[?25h")) // Show cursor

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		w.Write([]byte("\033[?12l\033[?25h")) // Show cursor
		// Stop notifying this channel
		signal.Stop(signals)
		// All Signals returned by the signal package should be of type syscall.Signal
		if signum, ok := s.(syscall.Signal); ok {
			syscall.Kill(syscall.Getpid(), signum)
		} else {
			panic(fmt.Sprintf("unexpected signal: %v", s))
		}
	}()

	mjpegs := NewMJPEGScanner(r)
	// for range time.Tick(time.Second / time.Duration(fps)) {
	// 	if !mjpegs.Scan() {
	// 		break
	// 	}
	ticker := time.Tick(time.Second / time.Duration(fps))
	for mjpegs.Scan(ticker) {
		img := mjpegs.Image()
		img = resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
		img = imaging.Invert(img)
		img = imaging.FlipH(img)
		if err := flush(w, img); err != nil {
			return err
		}
		select {
		case <-ticker:
		default:
		}
	}
	return mjpegs.Err()
}

func flush(w io.Writer, img image.Image) error {
	return nil
}

type MJPEGScanner struct {
	rdr io.Reader
	img image.Image
	err error
}

func NewMJPEGScanner(r io.Reader) *MJPEGScanner {
	return &MJPEGScanner{
		rdr: r,
	}
}

func (s *MJPEGScanner) Scan(ticker <-chan time.Time) bool {
	p := make([]byte, 1)
	var buf bytes.Buffer
	for {
		n, err := s.rdr.Read(p)
		if n == 0 {
			if err == nil {
				continue
			}
			if err != io.EOF {
				s.err = err
			}
			return false
		}

		if _, err := buf.Write(p); err != nil {
			s.err = err
			return false
		}

		if buf.Len() > 1 {
			data := buf.Bytes()
			if data[buf.Len()-2] == 0xff && data[buf.Len()-1] == 0xd9 {
				select {
				case <-ticker:
					img, err := jpeg.Decode(&buf)
					if err != nil {
						s.err = err
						return false
					}
					s.img = img
					return true
				default:
					buf.Truncate(0)
				}
			}
		}
	}
}

func (s *MJPEGScanner) Err() error {
	return s.err
}

func (s *MJPEGScanner) Image() image.Image {
	return s.img
}
