package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"syscall"
	"unsafe"

	"github.com/codegangsta/cli"
	"github.com/kevin-cantwell/dotmatrix"
	"github.com/nfnt/resize"
)

func main() {
	app := cli.NewApp()
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode Braille symbols."
	app.Flags = []cli.Flag{
		cli.Float64Flag{
			Name:  "l, luminosity",
			Usage: "Percentage (0.00-1.00) luminosity cutoff to determine filled or nofill pixels.",
		},
		cli.BoolFlag{
			Name:  "i, invert",
			Usage: "Invert filled/nofill pixels.",
		},
	}
	app.Action = func(c *cli.Context) {
		img, _, err := image.Decode(os.Stdin)
		if err != nil {
			exit(err.Error(), 1)
		}

		w, _, err := GetTerminalSize()
		if err != nil {
			exit(err.Error(), 1)
		}

		w = w * 2 // Since each symbol is two dots wide

		img = resize.Thumbnail(uint(w), uint(img.Bounds().Dy()), img, resize.NearestNeighbor)

		var opts []dotmatrix.ImageOpt
		if c.IsSet("luminosity") {
			opt := dotmatrix.WithLuminosity(float32(c.Float64("luminosity")))
			opts = append(opts, opt)
		}
		if c.IsSet("invert") {
			opt := dotmatrix.WithInvertedColors()
			opts = append(opts, opt)
		}
		enc := dotmatrix.NewImageEncoder(os.Stdout, opts...)
		if err := enc.Encode(img); err != nil {
			exit(err.Error(), 1)
		}
	}
	app.Run(os.Args)
}

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

func GetTerminalSize() (width, height int, err error) {
	var dimensions [4]uint16

	_, _, e := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&dimensions)),
		0, 0, 0,
	)
	if e != 0 {
		return -1, -1, e
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}
