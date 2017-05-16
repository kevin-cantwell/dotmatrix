package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	_ "golang.org/x/image/bmp"

	"github.com/codegangsta/cli"
	"github.com/disintegration/imaging"
	"github.com/kevin-cantwell/dotmatrix"
	"github.com/nfnt/resize"
	"golang.org/x/net/context"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.0.2"
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode braille symbols."
	app.UsageText = "1) dotmatrix [options] [file|url]\n" +
		/*      */ "   2) dotmatrix [options] < [file]"
	app.Author = "Kevin Cantwell"
	app.Email = "kevin.cantwell@gmail.com"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "invert,i",
			Usage: "Inverts black and white pixels.",
		},
		cli.StringFlag{
			Name:  "fit,f",
			Usage: "`W,H` = 80,25 scales down the image to fit a terminal size of 80 by 25.",
			Value: func() string {
				w, h, _ := getTerminalSize()
				return fmt.Sprintf("%d,%d", w, h)
			}(),
		},
		cli.Float64Flag{
			Name:  "gamma,g",
			Usage: "GAMMA less than 0 darkens the image and GAMMA greater than 0 lightens it.",
		},
		cli.Float64Flag{
			Name:  "brightness,b",
			Usage: "BRIGHTNESS = -100 gives solid black image. BRIGHTNESS = 100 gives solid white image.",
			Value: 0.0,
		},
		cli.Float64Flag{
			Name:  "contrast,c",
			Usage: "CONTRAST = -100 gives solid grey image. CONTRAST = 100 gives maximum contrast.",
			Value: 0.0,
		},
		cli.Float64Flag{
			Name:  "sharpen,s",
			Usage: "SHARPEN greater than 0 sharpens the image.",
			Value: 0.0,
		},
		cli.BoolFlag{
			Name:  "partymode,p",
			Usage: "Animates gifs in party mode.",
		},
		cli.BoolFlag{
			Name:  "mjpeg,m",
			Usage: "Processes input as an mjpeg stream.",
		},
	}
	app.Action = func(c *cli.Context) error {
		var reader io.Reader

		// Try to parse the args, if there are any, as a file or url
		if input := c.Args().First(); input != "" {
			// Is it a file?
			if file, err := os.Open(input); err == nil {
				reader = file
			} else {
				// Is it a url?
				resp, err := http.Get(input)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				reader = resp.Body
			}
		} else {
			reader = os.Stdin
		}

		// Tee out the reads while we attempt to decode the gif
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)

		// First try to play the input as an animated gif
		if giff, err := gif.DecodeAll(tee); err == nil {
			// Don't animate gifs with only a single frame
			if len(giff.Image) == 1 {
				if err := encodeImage(c, giff.Image[0]); err != nil {
					return err
				}
				return nil
			}
			// Animate
			if err := playGIF(c, giff, scalar(c, giff.Config.Width, giff.Config.Height)); err != nil {
				return err
			}
			return nil
		}

		// Assuming the gif decoing failed, copy the remaining bytes into the tee'd buffer
		go io.Copy(&buf, reader)

		if c.Bool("mjpeg") {
			return dotmatrix.PlayMJPEG(os.Stdout, &buf, 30)
		}

		// Now try to decode the image as static png/jpeg/gif
		img, _, err := image.Decode(&buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		// Encode image as a dotmatrix pattern
		return encodeImage(c, img)
	}

	if err := app.Run(os.Args); err != nil {
		exit(err.Error(), 1)
	}
}

func encodeImage(c *cli.Context, img image.Image) error {
	img = preprocessNonPaletted(c, img, scalar(c, img.Bounds().Dx(), img.Bounds().Dy()))
	return dotmatrix.Encode(os.Stdout, img)
}

func playGIF(c *cli.Context, giff *gif.GIF, scale float32) error {
	var w io.Writer = os.Stdout
	w.Write([]byte("\033[?25l")) // Hide cursor

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		cancel()
		w.Write([]byte("\033[0m"))            // Reset text color to default
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

	if len(giff.Image) == 1 {
		return encodeImage(c, giff.Image[0])
	}
	giff.Config = image.Config{
		Width:  int(float32(giff.Config.Width) * scale),
		Height: int(float32(giff.Config.Height) * scale),
	}
	for i, frame := range giff.Image {
		giff.Image[i] = preprocessPaletted(c, frame, scale)
	}
	if c.Bool("partymode") {
		w = &partyWriter{
			ctx:    ctx,
			writer: os.Stdout,
			colors: partyLights,
		}
	}
	return dotmatrix.PlayGIF(w, giff)
}

type partyWriter struct {
	ctx      context.Context
	writer   io.Writer
	colors   []int
	colorIdx int
}

var partyLights = []int{
	425,
	227,
	47,
	5, // Blue
	275,
	383,
	419,
	202,
	204,
}

func (w *partyWriter) Write(b []byte) (int, error) {
	if string(b) == "\033[0m" {
		if w.colorIdx >= len(w.colors) {
			w.colorIdx = 0
		}
		n, err := w.writer.Write([]byte(fmt.Sprintf("\033[38;5;%dm", w.colors[w.colorIdx])))
		w.colorIdx++
		select {
		case <-w.ctx.Done():
			w.writer.Write([]byte("\033[0m"))
		default:
		}
		return n, err
	}
	return w.writer.Write(b)
}

func scalar(c *cli.Context, w, h int) (scale float32) {
	defer func() {
		// Never scale larger, only smaller
		if scale > 1.0 {
			scale = 1.0
		}
	}()

	var cols, lines int
	if c.IsSet("fit") {
		parts := strings.Split(c.String("fit"), ",")
		if len(parts) != 2 {
			exit("fit option must be comma separated", 1)
		}
		cols, _ = strconv.Atoi(strings.Trim(parts[0], " "))
		lines, _ = strconv.Atoi(strings.Trim(parts[1], " "))
	}
	if cols == 0 && lines == 0 {
		var err error
		cols, lines, err = getTerminalSize()
		if err != nil {
			cols, lines = 80, 25 // Small, but a pretty standard default
		}
	}

	sx, sy := scalarX(cols, w), scalarY(lines, h)
	if sx == 0 {
		scale = sy
		return
	}
	if sy == 0 {
		scale = sx
		return
	}
	if sx < sy {
		scale = sx
		return
	}
	scale = sy
	return
}

// Multiply cols by 2 since each braille symbol is 2 pixels wide
func scalarX(cols int, dx int) float32 {
	if cols == 0 {
		return 0
	}
	return float32(cols*2) / float32(dx)
}

// Multiply lines by 4 since each braille symbol is 4 pixels high
func scalarY(lines int, dy int) float32 {
	if lines == 0 {
		return 0
	}
	return float32((lines-1)*4) / float32(dy)
}

func preprocessNonPaletted(c *cli.Context, img image.Image, scale float32) image.Image {
	return preprocessImage(c, img, scale)
}

func preprocessPaletted(c *cli.Context, img *image.Paletted, scale float32) *image.Paletted {
	processed := preprocessImage(c, img, scale)
	if processed == img {
		return img
	}

	// Create a new paletted image using a monochrome+transparent color palette.
	paletted := image.NewPaletted(processed.Bounds(), color.Palette{color.Black, color.White, color.Transparent})

	// If an image adjustment has occurred, we must redefine the bounds so that
	// we maintain the starting point. Not all images start at (0,0) after all.
	minX := img.Bounds().Min.X
	minY := img.Bounds().Min.Y
	offset := image.Pt(int(float32(minX)*scale), int(float32(minY)*scale))
	paletted.Rect = paletted.Bounds().Add(offset)
	// // Redraw the image with floyd steinberg image diffusion. This
	// // allows us to simulate gray or shaded regions with monochrome.
	draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), processed, processed.Bounds().Min)
	return paletted
}

func preprocessImage(c *cli.Context, img image.Image, scale float32) image.Image {
	width, height := uint(float32(img.Bounds().Dx())*scale), uint(float32(img.Bounds().Dy())*scale)
	img = resize.Resize(width, height, img, resize.NearestNeighbor)

	if c.IsSet("gamma") {
		gamma := c.Float64("gamma") + 1.0
		img = imaging.AdjustGamma(img, gamma)
	}
	if c.IsSet("brightness") {
		img = imaging.AdjustBrightness(img, c.Float64("brightness"))
	}
	if c.IsSet("sharpen") {
		img = imaging.Sharpen(img, c.Float64("sharpen"))
	}
	if c.IsSet("contrast") {
		img = imaging.AdjustContrast(img, c.Float64("contrast"))
	}
	if c.Bool("invert") {
		img = imaging.Invert(img)
	}

	return img
}

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

func getTerminalSize() (width, height int, err error) {
	var dimensions [4]uint16
	_, _, e := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stderr), // TODO: Figure out why we get "inappropriate ioctl for device" errors if we use stdin or stdout
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&dimensions)),
		0, 0, 0,
	)
	if e != 0 {
		return -1, -1, e
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}

type MJPEGScanner struct {
	rdr io.Reader
	// buf *bytes.Buffer
	img image.Image
	err error
}

func NewMJPEGScanner(r io.Reader) *MJPEGScanner {
	return &MJPEGScanner{
		rdr: r,
	}
}

func (s *MJPEGScanner) Scan() bool {
	var buf bytes.Buffer
	for {
		if _, err := io.CopyN(&buf, s.rdr, 1); err != nil {
			if err != io.EOF {
				s.err = err
			}
			return false
		}

		if buf.Len() > 1 {
			data := buf.Bytes()
			if data[buf.Len()-2] == 0xff && data[buf.Len()-1] == 0xd9 {
				s.img, s.err = jpeg.Decode(&buf)
				return true
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
