package main

import (
	"bufio"
	"context"
	"fmt"
	"image"
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

	"github.com/disintegration/imaging"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/kevin-cantwell/dotmatrix"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			showCursor(true)
			panic(r)
		}
	}()

	app := &cli.App{
		Name:            "dotmatrix",
		Usage:           "Render images and video as Unicode braille art",
		Version:         "0.1.1",
		UsageText:       "dotmatrix [options] [file|url]\ncat [file|url] | dotmatrix [options]",
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:               "invert",
				Aliases:            []string{"i"},
				Usage:              "Invert colors (for dark backgrounds)",
				DisableDefaultText: true,
			},
			&cli.Float64Flag{
				Name:        "gamma",
				Aliases:     []string{"g"},
				Usage:       "Adjust gamma: negative darkens, positive lightens",
				DefaultText: "-1.0 to 1.0",
			},
			&cli.Float64Flag{
				Name:        "brightness",
				Aliases:     []string{"b"},
				Usage:       "Adjust brightness",
				DefaultText: "-100 to 100",
			},
			&cli.Float64Flag{
				Name:        "contrast",
				Aliases:     []string{"c"},
				Usage:       "Adjust contrast",
				DefaultText: "-100 to 100",
			},
			&cli.Float64Flag{
				Name:        "sharpen",
				Aliases:     []string{"s"},
				Usage:       "Sharpen image",
				DefaultText: "0+",
			},
			&cli.BoolFlag{
				Name:               "mirror",
				Aliases:            []string{"m"},
				Usage:              "Flip image horizontally",
				DisableDefaultText: true,
			},
			&cli.BoolFlag{
				Name:               "mono",
				Usage:              "Disable Floyd-Steinberg dithering",
				DisableDefaultText: true,
			},
			&cli.BoolFlag{
				Name:               "webcam",
				Aliases:            []string{"w"},
				Usage:              "Capture from webcam (macOS only)",
				DisableDefaultText: true,
			},
			&cli.IntFlag{
				Name:        "framerate",
				Aliases:     []string{"fps"},
				Usage:       "Set playback framerate",
				Value:       -1,
				DefaultText: "native",
			},
			&cli.StringFlag{
				Name:    "mimeType",
				Aliases: []string{"mime"},
				Usage:   "Override auto-detected MIME type",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			go handleInterrupt(cancel)

			showCursor(false)
			defer showCursor(true)

			// Webcam mode doesn't need file/URL input
			if c.Bool("webcam") {
				return webcamAction(ctx, c, c.Int("framerate"))
			}

			reader, inputPath, mimeType, err := decodeReader(c)
			if err != nil {
				return err
			}

			if mime := c.String("mimeType"); mime != "" {
				mimeType = mime
			}

			switch mimeType {
			case "video/mp4", "application/mp4":
				if inputPath == "" {
					return fmt.Errorf("mp4: video playback requires a file path, not stdin or URL")
				}
				return mp4Action(ctx, c, inputPath, c.Int("framerate"))
			case "image/gif":
				return gifAction(ctx, c, reader)
			default:
				return imageAction(c, reader)
			}
		},
	}

	if err := app.Run(os.Args); err != nil {
		exit(err.Error(), 1)
	}
}

func handleInterrupt(cancel func()) {
	// Use buffered channel to avoid missing signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
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

func webcamAction(ctx context.Context, c *cli.Context, fps int) error {
	return dotmatrix.NewWebcamPrinter(os.Stdout, config(c)).Print(ctx, fps)
}

func mp4Action(ctx context.Context, c *cli.Context, inputPath string, fps int) error {
	return dotmatrix.NewMP4Printer(os.Stdout, config(c)).Print(ctx, inputPath, fps)
}

func decodeReader(c *cli.Context) (io.Reader, string, string, error) {
	var reader io.Reader = os.Stdin
	var inputPath string // Only set for local files (for MP4 support)

	// Assign to reader
	if input := c.Args().First(); input != "" {
		// Is it a file?
		if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
			file, err := os.Open(input)
			if err != nil {
				return nil, "", "", err
			}
			reader = file
			inputPath = input // Store the file path for MP4 support
		} else {
			// Is it a url?
			resp, err := http.Get(input)
			if err != nil {
				return nil, "", "", err
			}
			reader = resp.Body
		}
	}

	bufioReader := bufio.NewReader(reader)

	peeked, err := bufioReader.Peek(512)
	if err != nil {
		return nil, "", "", err
	}

	mimeType := http.DetectContentType(peeked)

	return bufioReader, inputPath, mimeType, nil
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
	// Mirror flips the image on its vertical axis
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

	width := int(f.scale * float64(img.Bounds().Dx()))
	height := int(f.scale * float64(img.Bounds().Dy()))
	return imaging.Resize(img, width, height, imaging.NearestNeighbor)
}

func terminalDimensions() (int, int) {
	var cols, rows int

	if term.IsTerminal(int(os.Stdout.Fd())) {
		tw, th, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			th-- // Accounts for the terminal prompt
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
