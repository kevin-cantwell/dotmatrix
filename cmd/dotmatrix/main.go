package main

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "golang.org/x/image/bmp"

	"github.com/codegangsta/cli"
	"github.com/disintegration/imaging"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/kevin-cantwell/dotmatrix"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			showCursor(true)
			panic(r)
		}
	}()

	// print a gopher after the help menu
	defaultHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		defaultHelpPrinter(w, templ, data)
		config := &dotmatrix.Config{
			Filter: &Filter{Invert: false},
			Drawer: draw.FloydSteinberg,
		}
		dotmatrix.NewPrinter(os.Stdout, config).Print(gopher())
	}

	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode braille symbols."
	app.UsageText = "1) dotmatrix [options] [file|url]\n" +
		/*      */ "   2) dotmatrix [options] < [file]"
	app.Author = "Kevin Cantwell"
	app.Email = "kevin.cantwell@gmail.com"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "invert,i",
			Usage: "Inverts image color. Useful for black background terminals",
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
			Name:  "mirror,m",
			Usage: "Mirrors the image.",
		},
		cli.BoolFlag{
			Name:  "mono",
			Usage: "Images are drawn without Floyd Steinberg diffusion.",
		},
		cli.BoolFlag{
			Name:  "motion,mjpeg",
			Usage: "Interpret input as an mjpeg stream, such as from a webcam.",
		},
		cli.IntFlag{
			Name:  "framerate,fps",
			Usage: "Force a framerate for mjpeg streams. Default is -1 (ie: no delay between frames).",
			Value: -1,
		},
		cli.StringFlag{
			Name:  "mimeType,mime",
			Usage: "Force interpretation of a specific mime type (eg: \"image/gif\". Default is to examine the first 512 bytes and make an educated guess.",
		},
	}
	app.Action = func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		go handleInterrupt(cancel)

		showCursor(false)
		defer showCursor(true)

		reader, mimeType, err := decodeReader(c)
		if err != nil {
			return err
		}

		if mime := c.String("mimeType"); mime != "" {
			mimeType = mime
		}

		if c.Bool("motion") {
			return mjpegAction(ctx, c, reader, c.Int("framerate"))
		}

		switch mimeType {
		case "video/x-motion-jpeg":
			return mjpegAction(ctx, c, reader, c.Int("framerate"))
		case "image/gif":
			return gifAction(ctx, c, reader)
		default:
			return imageAction(c, reader)
		}
	}

	if err := app.Run(os.Args); err != nil {
		exit(err.Error(), 1)
	}
}

func handleInterrupt(cancel func()) {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signals
		showCursor(true)
		// Stop notifying this channel
		signal.Stop(signals)
		cancel()

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

func showCursor(show bool) {
	if show {
		fmt.Fprint(os.Stdout, "\033[?12l\033[?25h")
	} else {
		fmt.Fprint(os.Stdout, "\033[?25l")
	}
}

func config(c *cli.Context) *dotmatrix.Config {
	return &dotmatrix.Config{
		Filter: &Filter{
			Gamma:      c.Float64("gamma"),
			Brightness: c.Float64("brightness"),
			Contrast:   c.Float64("contrast"),
			Sharpen:    c.Float64("sharpen"),
			Invert:     c.Bool("invert"),
			Mirror:     c.Bool("mirror"),
		},
		Drawer: func() draw.Drawer {
			if c.Bool("mono") {
				return draw.Src
			}
			return draw.FloydSteinberg
		}(),
	}
}

func imageAction(c *cli.Context, r io.Reader) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	return dotmatrix.NewPrinter(os.Stdout, config(c)).Print(img)
}

func gifAction(ctx context.Context, c *cli.Context, r io.Reader) error {
	giff, err := gif.DecodeAll(r)
	if err != nil {
		return err
	}
	return dotmatrix.NewGIFPrinter(os.Stdout, config(c)).Print(ctx, giff)
}

func mjpegAction(ctx context.Context, c *cli.Context, r io.Reader, fps int) error {
	return dotmatrix.NewMJPEGPrinter(os.Stdout, config(c)).Print(ctx, r, fps)
}

func decodeReader(c *cli.Context) (io.Reader, string, error) {
	var reader io.Reader = os.Stdin

	// Assign to reader
	if input := c.Args().First(); input != "" {
		// Is it a file?
		if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
			file, err := os.Open(input)
			if err != nil {
				return nil, "", err
			}
			reader = file
		} else {
			// Is it a url?
			if resp, err := http.Get(input); err != nil {
				return nil, "", err
			} else {
				reader = resp.Body
			}
		}
	}

	bufioReader := bufio.NewReader(reader)

	peeked, err := bufioReader.Peek(512)
	if err != nil {
		return nil, "", err
	}

	mimeType := http.DetectContentType(peeked)

	return bufioReader, mimeType, nil
}

type Filter struct {
	// Gamma less than 0 darkens the image and GAMMA greater than 0 lightens it.
	Gamma float64
	// Brightness = -100 gives solid black image. Brightness = 100 gives solid white image.
	Brightness float64
	// Contrast = -100 gives solid grey image. Contrast = 100 gives maximum contrast.
	Contrast float64
	// Sharpen greater than 0 sharpens the image.
	Sharpen float64
	// Inverts pixel color. Transparent pixels remain transparent.
	Invert bool
	// Mirror flips the image on it's vertical axis
	Mirror bool

	scale float64
}

func (f *Filter) Filter(img image.Image) image.Image {
	if f.Gamma != 0 {
		img = imaging.AdjustGamma(img, f.Gamma+1.0)
	}
	if f.Brightness != 0 {
		img = imaging.AdjustBrightness(img, f.Brightness)
	}
	if f.Sharpen != 0 {
		img = imaging.Sharpen(img, f.Sharpen)
	}
	if f.Contrast != 0 {
		img = imaging.AdjustContrast(img, f.Contrast)
	}
	if f.Mirror {
		img = imaging.FlipH(img)
	}
	if f.Invert {
		img = imaging.Invert(img)
	}

	// Only calculate the scalar values once because gifs
	if f.scale == 0 {
		cols, rows := terminalDimensions()
		dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
		scale := scalar(dx, dy, cols, rows)
		if scale >= 1.0 {
			scale = 1.0
		}
		f.scale = scale
	}

	width := uint(f.scale * float64(img.Bounds().Dx()))
	height := uint(f.scale * float64(img.Bounds().Dy()))
	return resize.Resize(width, height, img, resize.NearestNeighbor)
}

func terminalDimensions() (int, int) {
	var cols, rows int

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		tw, th, err := terminal.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			th -= 1 // Accounts for the terminal prompt
			if cols == 0 {
				cols = tw
			}
			if rows == 0 {
				rows = th
			}
		}
	}

	// Small, but fairly standard defaults
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 25
	}

	return cols, rows
}

func scalar(dx, dy int, cols, rows int) float64 {
	scale := float64(1.0)
	scaleX := float64(cols*2) / float64(dx)
	scaleY := float64(rows*4) / float64(dy)

	if scaleX < scale {
		scale = scaleX
	}
	if scaleY < scale {
		scale = scaleY
	}

	return scale
}

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⢀⢀⡀⡄⡄⠤⣄⡠⢄⠤⢠⢀⠄⡀⢀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⠠⡄⡆⠮⢕⢕⢍⡢⢇⠧⢍⢇⢖⠬⡪⠭⢡⠣⡇⢫⢕⢕⠍⡦⠆⡤⠀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀⡄⡆⢗⡪⣑⡱⡸⣩⢪⠒⡕⡬⠥⡫⡒⢪⢜⢔⡱⠭⡱⠥⢣⢇⢎⢎⢪⡒⢭⠪⢭⢒⡣⢆⡄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠠⡢⢕⡣⢕⢭⠑⠒⠑⠈⠘⠄⠇⠭⡜⢌⢖⢕⢜⢕⣊⢔⡒⡭⠜⡍⣒⢕⠥⠥⠃⠚⠈⠉⠸⠘⢨⢆⠳⠌⡧⢄⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡠⡢⢕⢍⡪⢆⡓⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⠸⡢⠕⡕⢅⡆⢗⢜⠬⢕⡪⡪⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠋⡭⡪⣒⣒⣒⣂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⢸⠸⢌⡲⣑⡌⠇⠀⠀⠀⠀⠀⠀⠀⣠⣤⣦⣤⣀⡀⠨⢍⢎⢖⢕⢪⠪⢒⠥⢎⠀⠀⠀⠀⠀⠀⠀⣀⣤⣴⣤⣄⡀⠈⢎⡰⣒⢂⢖⢝⢔⠄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⢰⢩⢕⢕⠭⢜⣘⢔⢪⠀⠀⠀⠀⠀⠀⢀⣾⣿⣿⡿⠟⠻⢷⡀⠪⡪⣂⠮⢜⠜⡕⠭⠁⠀⠀⠀⠀⠀⠀⣼⣿⣿⣿⠟⠟⢿⡄⢨⢪⢒⣊⢎⢎⣨⢒⡣⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡠⡕⡕⡕⠕⢕⢜⣒⡸⡢⠭⠀⠀⠀⠀⠀⠀⢘⣿⣿⣿⣇⠀⠀⣸⡇⠨⡪⡢⠭⣒⡚⢬⡚⠄⠀⠀⠀⠀⠀⠀⣿⣿⣿⣧⡀⠀⣐⡇⠐⡕⠕⣒⠪⣰⢸⠸⢘⠬⠥⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⢄⠪⡁⢣⠩⡰⡠⡜⣒⢕⠕⡒⡍⡎⡖⢢⢜⠬⢑⠅⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣿⡿⠟⠀⡪⡪⡪⡪⣂⢗⢊⡪⣂⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣿⡿⠟⠀⠬⡪⣚⠤⢫⢢⠥⢫⠥⢫⠱⡪⠢⡢⠅⣃⢃⠣⡑⠠⡀⠀⠀
// ⠀⠀⠀⠀⡐⢩⢂⣡⡪⣜⣆⣖⠕⢮⢒⣊⢖⣃⠕⡕⡕⢪⢜⢜⠭⢔⠄⠀⠀⠀⠀⠀⠀⠀⠈⠉⠁⠀⠀⡬⣘⣬⣮⣮⣶⣵⣕⣖⢕⢄⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠀⢀⠬⢹⢌⡒⣎⡱⣑⡩⢕⢕⣸⠸⡜⢎⢧⢬⣢⡣⣜⢔⠡⡔⡀⠀
// ⠀⠀⠀⠀⡸⡐⡡⣳⣝⣗⣗⢷⢝⣑⢕⡜⣒⠬⣍⣊⢎⠥⣃⡣⡱⡪⠭⡅⠤⢀⠀⠀⠀⠀⠀⡀⡀⡤⣫⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣾⡢⠤⡀⠀⠀⠀⠀⠀⢀⢀⢰⢱⢉⢇⢇⠖⠕⡬⠔⡪⡪⡒⣒⡪⡋⣞⢬⣻⣪⢯⣳⣝⡂⡒⡀⠀
// ⠀⠀⠀⠀⡒⠨⠌⣳⡳⣵⣳⣫⡳⣑⢕⠜⣒⡢⢕⣂⢎⢖⢕⡰⠥⡣⢩⠪⡹⡘⡜⢲⢡⢕⣊⢎⣒⣱⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡱⡩⠥⣋⠬⡢⢍⢲⣉⢎⢎⢪⠪⢬⠪⠭⠜⡍⣊⠎⢎⣒⠜⣎⢴⢕⢷⣝⣗⢗⠗⡄⢣⠈⠀
// ⠀⠀⠀⠀⠈⠑⡑⢔⢘⠬⡪⣔⢼⡘⠥⡫⡒⣒⠥⣊⢎⠎⣒⢜⡱⡩⠕⡥⡣⡱⣉⢇⣃⢥⢒⡡⣆⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢧⡍⡇⡜⣱⡘⡕⢕⠬⢔⡪⡪⡪⠥⡪⢍⢇⢎⢖⠭⢑⢜⣒⠪⢦⠫⡸⢜⠨⡘⠄⠎⠔⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠑⠐⢘⢖⠪⣒⢜⢍⢪⠪⠒⣎⢢⠣⢕⢕⢕⠬⠜⡜⡌⡎⣎⢢⠓⡤⡣⣵⣳⡳⣝⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⣗⣗⣗⡵⣑⠴⡱⡡⢫⠥⡪⡪⠌⡣⠭⡜⠴⢑⢎⢎⠭⢒⡢⠭⡒⡭⠝⡜⠈⠈⠁⠁⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠐⡕⢣⠣⣑⢕⢕⠭⠍⡲⡱⡩⢕⠥⠕⡜⢍⣊⡜⡌⡖⢊⠞⡔⣻⣺⡺⣺⣳⣳⣻⣻⣿⣿⣿⣿⣿⣿⡿⣟⣗⣗⣗⣗⡵⡯⡧⢱⠒⡭⡑⣕⠒⡕⠭⠭⠴⢩⢍⠮⡜⠤⡫⡪⡪⢱⢪⠪⠭⠜⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠨⡚⢌⢳⢸⢸⢠⠮⢙⡒⡕⢕⢜⢜⢕⠼⡸⢄⠧⡜⢜⢕⡚⡜⡵⣳⣝⢷⢵⡳⣵⣳⡳⡽⡽⡽⡽⣺⣝⣗⣗⣗⣗⣗⡽⡽⢝⢕⢕⢕⡸⢤⢫⠪⠭⢜⢕⢕⠕⢕⡘⡭⠜⡔⡎⢱⠪⡑⡭⠍⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡸⡉⣎⡒⢎⡢⣍⠳⢸⢨⢪⠒⡕⠪⡢⡓⡎⢱⢒⣡⢣⠲⢱⢉⠗⡹⢽⢵⣻⣺⣺⡺⡽⡽⣝⡽⣵⣳⡳⣵⡳⣵⡳⡝⡍⢖⢕⢡⢕⡒⢥⠣⡱⢍⢎⣊⠦⠭⣑⢎⢪⠪⡜⠬⡕⠎⡇⣃⢏⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠨⣊⢎⢒⠬⡪⡪⡰⡪⡣⡱⡑⡇⡭⢍⡪⡪⡪⡱⡑⡜⢌⠭⣑⢕⢍⠆⠀⠉⠘⠘⠚⠚⠝⠝⠮⠛⠚⠚⠚⠑⠉⠀⠀⢪⠪⢜⡒⡱⣑⠲⡅⣇⣓⢪⢲⢘⣂⢏⣔⠪⡜⢲⠩⠕⡜⢍⡜⠜⡤⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣠⠓⡎⡭⢌⡪⡪⡊⡬⡪⡪⡪⠪⢢⢣⠣⠜⡬⡪⣚⡨⢍⡢⢕⠕⡅⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢕⠭⢑⣒⢕⢕⠕⡕⡔⣒⡱⡡⠕⣒⣊⢆⡕⠭⣡⠫⣉⢎⢖⢩⠭⠜⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⢨⠒⡍⡎⡪⡪⣊⣊⠎⢎⠬⢔⡒⡭⠣⡣⢱⠭⡪⠬⣒⡨⢣⢜⠱⡱⣂⠀⠀⠀⠀⠀⠀⠀⠀⠆⠀⠀⠀⠀⠀⠀⠀⢀⢕⠭⣑⢜⠤⡣⡱⡱⡑⣒⢜⠬⢹⠰⡢⡣⡘⡍⡢⢕⠣⢣⣑⢕⢍⡜⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⢞⢒⢱⠱⡸⢄⢖⢍⢱⡙⡬⡪⠪⡚⠬⡕⢅⢖⠭⢔⣊⠥⡣⣓⢪⢒⡢⡠⢄⢄⢄⢄⠤⢌⢇⢆⠤⢀⢄⢄⡠⡐⡕⢕⢕⠕⠕⡕⢪⠪⣊⡪⡒⡪⡪⡣⡱⣑⣊⢕⠥⢫⠥⢫⠥⢒⢕⢕⠌⠀⠀⠀⠀⠀⠀⠀
// ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠁⠀⠈⠀⠀⠁⠀⠀⠈⠀⠈⠀⠀⠁⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠀⠈⠀⠀⠀⠈⠀⠁⠀⠀⠈⠀⠀⠀⠈⠀⠁⠀⠈⠀⠀⠀⠈⠀⠀⠀⠀⠀⠀⠈⠀⠈⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
func gopher() image.Image {
	// Initialize the graphic context on an RGBA image
	dest := image.NewRGBA(image.Rect(0, 0, 297, 148.0))
	gc := draw2dimg.NewGraphicContext(dest)

	gc.SetStrokeColor(image.Black)
	gc.SetFillColor(image.White)
	gc.Save()
	// Draw a (partial) gopher
	gc.Translate(-15, -45)

	var x, y, w, h float64 = 48, 48, 240, 72

	h23 := (h * 2) / 3

	blf := color.RGBA{0, 0, 0, 0xff}          // black
	wf := color.RGBA{0xff, 0xff, 0xff, 0xff}  // white
	nf := color.RGBA{0x8B, 0x45, 0x13, 0xff}  // brown opaque
	brf := color.RGBA{0x8B, 0x45, 0x13, 0x99} // brown transparant
	brb := color.RGBA{0x8B, 0x45, 0x13, 0xBB} // brown transparant

	// round head top
	gc.MoveTo(x, y+h*1.002)
	gc.CubicCurveTo(x+w/4, y-h/3, x+3*w/4, y-h/3, x+w, y+h*1.002)
	gc.Close()
	gc.SetFillColor(brb)
	gc.Fill()

	// rectangle head bottom
	draw2dkit.RoundedRectangle(gc, x, y+h, x+w, y+h+h, h/5, h/5)
	gc.Fill()

	// left ear outside
	draw2dkit.Circle(gc, x, y+h, w/12)
	gc.SetFillColor(brf)
	gc.Fill()

	// left ear inside
	draw2dkit.Circle(gc, x, y+h, 0.5*w/12)
	gc.SetFillColor(nf)
	gc.Fill()

	// right ear outside
	draw2dkit.Circle(gc, x+w, y+h, w/12)
	gc.SetFillColor(brf)
	gc.Fill()

	// right ear inside
	draw2dkit.Circle(gc, x+w, y+h, 0.5*w/12)
	gc.SetFillColor(nf)
	gc.Fill()

	// left eye outside white
	draw2dkit.Circle(gc, x+w/3, y+h23, w/9)
	gc.SetFillColor(wf)
	gc.Fill()

	// left eye black
	draw2dkit.Circle(gc, x+w/3+w/24, y+h23, 0.5*w/9)
	gc.SetFillColor(blf)
	gc.Fill()

	// left eye inside white
	draw2dkit.Circle(gc, x+w/3+w/24+w/48, y+h23, 0.2*w/9)
	gc.SetFillColor(wf)
	gc.Fill()

	// right eye outside white
	draw2dkit.Circle(gc, x+w-w/3, y+h23, w/9)
	gc.Fill()

	// right eye black
	draw2dkit.Circle(gc, x+w-w/3+w/24, y+h23, 0.5*w/9)
	gc.SetFillColor(blf)
	gc.Fill()

	// right eye inside white
	draw2dkit.Circle(gc, x+w-(w/3)+w/24+w/48, y+h23, 0.2*w/9)
	gc.SetFillColor(wf)
	gc.Fill()

	// left tooth
	gc.SetFillColor(wf)
	draw2dkit.RoundedRectangle(gc, x+w/2-w/8, y+h+h/2.5, x+w/2-w/8+w/8, y+h+h/2.5+w/6, w/10, w/10)
	gc.Fill()

	// right tooth
	draw2dkit.RoundedRectangle(gc, x+w/2, y+h+h/2.5, x+w/2+w/8, y+h+h/2.5+w/6, w/10, w/10)
	gc.Fill()

	// snout
	draw2dkit.Ellipse(gc, x+(w/2), y+h+h/2.5, w/6, w/12)
	gc.SetFillColor(nf)
	gc.Fill()

	// nose
	draw2dkit.Ellipse(gc, x+(w/2), y+h+h/7, w/10, w/12)
	gc.SetFillColor(blf)
	gc.Fill()

	gc.Restore()

	return dest
}
