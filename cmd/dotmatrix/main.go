package main

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	_ "golang.org/x/image/bmp"

	"github.com/codegangsta/cli"
	"github.com/disintegration/imaging"
	"github.com/kevin-cantwell/dotmatrix"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
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
			Usage: "Inverts image color.",
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
			Name:  "camera,cam",
			Usage: "Use FaceTime camera input (Requires ffmpeg+avfoundation).",
		},
		cli.BoolFlag{
			Name:  "video,vid",
			Usage: "Use video input (Requires ffmpeg).",
		},
		cli.BoolFlag{
			Name:  "mono",
			Usage: "If specified, image is drawn without Floyd Steinberg diffusion",
		},
	}
	app.Action = func(c *cli.Context) error {
		reader, mediaType, err := decodeReader(c)
		if err != nil {
			return err
		}
		defer reader.Close()

		config := &dotmatrix.Config{
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

		switch mediaType {
		case "mjpeg":
			return dotmatrix.NewMJPEGAnimator(os.Stdout, config).Animate(reader, 30)
		case "gif":
			giff, err := gif.DecodeAll(reader)
			if err != nil {
				return err
			}
			return dotmatrix.NewGIFAnimator(os.Stdout, config).Animate(giff)
		default:
			img, _, err := image.Decode(reader)
			if err != nil {
				return err
			}
			return dotmatrix.NewEncoder(os.Stdout, config).Encode(img)
		}
	}

	if err := app.Run(os.Args); err != nil {
		exit(err.Error(), 1)
	}
}

func decodeReader(c *cli.Context) (io.ReadCloser, string, error) {
	// Are we reading from isight?
	if c.Bool("camera") {
		cmd := exec.Command("ffmpeg", "-r", "30", "-f", "avfoundation", "-i", "FaceTime", "-f", "mjpeg", "-loglevel", "panic", "pipe:")
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, "", err
		}
		if err := cmd.Start(); err != nil {
			return nil, "", err
		}
		go func() {
			if err := cmd.Wait(); err != nil {
				exit(err.Error(), 1)
			}
		}()
		return stdoutPipe, "mjpeg", nil
	}

	// Get the input argument
	input := c.Args().First()
	if input == "" {
		return nil, "", errors.New("dotmatrix: no input specified")
	}

	// What's the media type?
	var mediaType string
	switch {
	case strings.HasSuffix(strings.ToLower(input), ".gif"):
		mediaType = "gif"
	case strings.HasSuffix(strings.ToLower(input), ".mjpeg"):
		mediaType = "mjpeg"
	default:
		mediaType = "image"
	}

	// Is it a url?
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		if resp, err := http.Get(input); err != nil {
			return nil, "", err
		} else {
			return resp.Body, mediaType, nil
		}
	}

	// Is it a file?
	file, err := os.Open(input)
	return file, mediaType, err
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
