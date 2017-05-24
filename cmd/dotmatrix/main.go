package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
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

		drawer := draw.FloydSteinberg
		if c.Bool("mono") {
			drawer = draw.Src
		}

		filter := Filter{
			Gamma:      c.Float64("gamma"),
			Brightness: c.Float64("brightness"),
			Contrast:   c.Float64("contrast"),
			Sharpen:    c.Float64("sharpen"),
			Invert:     c.Bool("invert"),
			Mirror:     c.Bool("mirror"),
			Drawer:     drawer,
		}

		switch mediaType {
		case "mjpeg":
			return dotmatrix.NewMJPEGAnimator(os.Stdout, drawer, nil).Animate(reader, 30)
		case "gif":
			giff, err := gif.DecodeAll(reader)
			if err != nil {
				return err
			}
			// giff = pre.ProcessGIF(giff)
			return dotmatrix.NewGIFAnimator(os.Stdout, filter, nil).Animate(giff)
		default:
			img, _, err := image.Decode(reader)
			if err != nil {
				return err
			}
			// img = pre.ProcessImage(img)
			return dotmatrix.NewEncoder(os.Stdout, filter).Encode(img)
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

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

type PreProcessor struct {
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
	// Drawer is the algorithm for converting the image to monochrome
	Drawer draw.Drawer
}

func (p *PreProcessor) ProcessImage(img image.Image) image.Image {
	cols, rows := p.terminalDimensions()
	dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
	scale := p.scalar(dx, dy, cols, rows)
	img = p.resize(img, scale)
	return p.adjust(img)
}

func (p *PreProcessor) ProcessGIF(giff *gif.GIF) *gif.GIF {
	cols, rows := p.terminalDimensions()
	dx, dy := giff.Config.Width, giff.Config.Height
	scale := p.scalar(dx, dy, cols, rows)

	newGiff := &gif.GIF{
		Config: image.Config{
			Width:  int(float64(giff.Config.Width) * scale),
			Height: int(float64(giff.Config.Height) * scale),
		},
		Delay:           giff.Delay,
		LoopCount:       giff.LoopCount,
		Disposal:        giff.Disposal,
		BackgroundIndex: giff.BackgroundIndex,
	}

	monoPallette := []color.Color{color.Black, color.White, color.Transparent}

	// Redraw each frame of the gif to match the options
	for _, frame := range giff.Image {
		img := p.resize(frame, scale)
		img = p.adjust(img)

		// Create a new paletted image using gif's color palette.
		// If an image adjustment has occurred, we must redefine the bounds so that
		// we maintain the starting point. Not all images start at (0,0) after all.
		paletted := image.NewPaletted(img.Bounds(), monoPallette)
		minX := frame.Bounds().Min.X // Important to use the original frame mins here!
		minY := frame.Bounds().Min.Y
		offset := image.Pt(int(float64(minX)*scale), int(float64(minY)*scale))
		paletted.Rect = paletted.Bounds().Add(offset)

		p.Drawer.Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)

		newGiff.Image = append(newGiff.Image, paletted)
	}

	return newGiff
}

func (p *PreProcessor) resize(img image.Image, scale float64) image.Image {
	if scale >= 1.0 {
		return img
	}
	dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
	width := uint(scale * float64(dx))
	height := uint(scale * float64(dy))
	return resize.Resize(width, height, img, resize.NearestNeighbor)
}

func (p *PreProcessor) adjust(img image.Image) image.Image {
	if p.Gamma != 0 {
		img = imaging.AdjustGamma(img, p.Gamma+1.0)
	}
	if p.Brightness != 0 {
		img = imaging.AdjustBrightness(img, p.Brightness)
	}
	if p.Sharpen != 0 {
		img = imaging.Sharpen(img, p.Sharpen)
	}
	if p.Contrast != 0 {
		img = imaging.AdjustContrast(img, p.Contrast)
	}
	if p.Mirror {
		img = imaging.FlipH(img)
	}
	if p.Invert {
		img = imaging.Invert(img)
	}
	return img
}

func (p *PreProcessor) scalar(dx, dy int, cols, rows int) float64 {
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

func (p *PreProcessor) terminalDimensions() (int, int) {
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
	// Drawer is the algorithm used for drawing the image into a monochrome palette
	Drawer draw.Drawer
}

func (f Filter) Filter(img image.Image) image.Image {
	// Adjust
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

	// Resize
	cols, rows := terminalDimensions()
	dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
	scale := scalar(dx, dy, cols, rows)
	if scale >= 1.0 {
		return img
	}
	width := uint(scale * float64(dx))
	height := uint(scale * float64(dy))
	return resize.Resize(width, height, img, resize.NearestNeighbor)

	return img
}

func (f Filter) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
	drawer := f.Drawer
	if f.Drawer == nil {
		drawer = draw.FloydSteinberg
	}
	drawer.Draw(dst, r, src, sp)
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
