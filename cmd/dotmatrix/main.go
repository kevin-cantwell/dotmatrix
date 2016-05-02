package main

import (
	"bytes"
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
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	_ "golang.org/x/image/bmp"

	"github.com/codegangsta/cli"
	"github.com/disintegration/imaging"
	"github.com/kevin-cantwell/dotmatrix"
	"github.com/nfnt/resize"
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
	}
	app.Action = func(c *cli.Context) {
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
					exit(err.Error(), 1)
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
					exit(err.Error(), 1)
				}
				return
			}
			// Animate
			if err := playGIF(c, giff, scalar(c, giff.Image[0])); err != nil {
				exit(err.Error(), 1)
			}
			return
		}

		// Copy the remaining bytes into the buffer
		io.Copy(&buf, reader)
		// Now try to decode the image as static png/jpeg/gif
		img, _, err := image.Decode(&buf)
		if err != nil {
			exit(err.Error(), 1)
		}
		// Encode image as a dotmatrix pattern
		if err := encodeImage(c, img); err != nil {
			exit(err.Error(), 1)
		}
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func encodeImage(c *cli.Context, img image.Image) error {
	img = preprocessImage(c, img, scalar(c, img))
	return dotmatrix.Encode(os.Stdout, img)
}

func playGIF(c *cli.Context, giff *gif.GIF, scale float32) error {
	if len(giff.Image) == 1 {
		return encodeImage(c, giff.Image[0])
	}
	giff.Config = image.Config{
		Width:  int(float32(giff.Config.Width) * scale),
		Height: int(float32(giff.Config.Height) * scale),
	}
	for i, frame := range giff.Image {
		giff.Image[i] = preprocessImage(c, frame, scale)
	}
	return dotmatrix.PlayGIF(os.Stdout, giff)
}

func scalar(c *cli.Context, img image.Image) float32 {
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

	var scalar float32

	// Multiply cols by 2 since each braille symbol is 2 pixels wide
	// Multiply lines by 4 since each braille symbol is 4 pixels high
	width, height := float32(cols*2), float32((lines-1)*4)
	scalarX, scalarY := width/float32(img.Bounds().Dx()), height/float32(img.Bounds().Dy())
	if scalarX == 0 {
		scalar = scalarY
	}
	if scalarY == 0 {
		scalar = scalarX
	}
	if scalarX < scalarY {
		scalar = scalarX
	} else {
		scalar = scalarY
	}
	if scalar > 1.0 {
		return 1.0
	}
	return scalar
}

func preprocessImage(c *cli.Context, img image.Image, scale float32) *image.Paletted {
	orig := img // Save off since image adjustment resets bounds min to ZP

	width, height := uint(float32(img.Bounds().Dx())*scale), uint(float32(img.Bounds().Dy())*scale)
	img = resize.Thumbnail(width, height, img, resize.NearestNeighbor)

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

	// Create a new paletted image using a monochrome+transparent color palette.
	paletted := image.NewPaletted(img.Bounds(), []color.Color{color.Black, color.White, color.Transparent})
	// If an image adjustment has occurred, we must redefine the bounds so that
	// we maintain the starting point. Not all images start at (0,0) after all.
	if orig != img {
		offset := image.Pt(int(float32(orig.Bounds().Min.X)*scale), int(float32(orig.Bounds().Min.Y)*scale))
		paletted.Rect = paletted.Bounds().Add(offset)
	}
	// Redraw the image with floyd steinberg image diffusion. This
	// allows us to simulate gray or shaded regions with monochrome.
	draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)
	return paletted
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
