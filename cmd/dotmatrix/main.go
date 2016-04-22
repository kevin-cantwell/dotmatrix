package main

import (
	"errors"
	"fmt"
	"image"
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

	"github.com/codegangsta/cli"
	"github.com/kevin-cantwell/dotmatrix"
	"github.com/nfnt/resize"
)

func main() {
	cols, rows, err := getTerminalSize()
	if err != nil {
		cols, rows = 80, 25 // Small, but a pretty standard default
	}

	var dimensions *string

	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode braille symbols."
	app.UsageText = "1) dotmatrix [options] [file|url]\n" +
		/*      */ "   2) dotmatrix [options] < [file]"
	app.Author = "Kevin Cantwell"
	app.Email = "kevin.cantwell@gmail.com"
	app.Flags = []cli.Flag{
		cli.Float64Flag{
			Name:  "luminosity,l",
			Usage: "Percentage value, between 0 (all black) and 1 (all white). Defaults to 0.5.",
			Value: 0.5,
		},
		cli.BoolFlag{
			Name:  "invert,i",
			Usage: "Inverts colors.",
		},
		cli.StringFlag{
			Name:        "dimensions,d",
			Destination: dimensions,
			// This function achieves a specific goal: to only call getTerminalSize()
			// if this flag is unset while allowing a pretty help output.
			Value: func() string {
				if dimensions == nil {
					cols, rows, err := getTerminalSize()
					if err != nil {
						cols, rows = 80, 25 // Small, but a pretty standard default
					}
					d := fmt.Sprintf("%d,%d", cols, rows)
					dimensions = &d
				}
				return *dimensions
			}(),
			Usage: "Comma-delimited width and height of output. The default output is constrained by the terminal size.",
		},
		cli.BoolFlag{
			Name:  "play,p",
			Usage: "Animates gifs in a pseudo-graphical user interface (ESC or CTRL-C to quit).",
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

		config := dotmatrix.Config{
			Luminosity: float32(c.Float64("luminosity")),
			Inverted:   c.Bool("invert"),
		}

		if c.Bool("play") {
			giff, err := gif.DecodeAll(reader)
			if err != nil {
				exit(err.Error(), 1)
			}
			enc := dotmatrix.NewGIFEncoder(config)
			if err := enc.Encode(os.Stdout, giff); err != nil {
				exit(err.Error(), 1)
			}
			return
		}

		img, _, err := image.Decode(reader)
		if err != nil {
			exit(err.Error(), 1)
		}

		// Calculate the width and height of the output image
		cols, rows, err = parseDimensions(*dimensions)
		if err != nil {
			exit(err.Error(), 1)
		}
		// Multiply by 2 since each braille symbol is 2 pixels wide
		width := cols * 2
		// Multiply by 4 since each braille symbol is 4 pixels high
		height := (rows - 1) * 4

		// Resize to fit
		if width == 0 {
			width = img.Bounds().Dx()
		}
		if height == 0 {
			height = img.Bounds().Dy()
		}
		img = resize.Thumbnail(uint(width), uint(height), img, resize.NearestNeighbor)

		enc := dotmatrix.NewImageEncoder(config)
		if err := enc.Encode(os.Stdout, img); err != nil {
			exit(err.Error(), 1)
		}
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

func parseDimensions(dim string) (int, int, error) {
	parts := strings.Split(dim, ",")
	if len(parts) != 2 {
		return 0, 0, errors.New("dotmatrix: dimensions must be of the form \"W,H\"")
	}
	w, err := strconv.Atoi(strings.Trim(parts[0], " "))
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(strings.Trim(parts[1], " "))
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
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
