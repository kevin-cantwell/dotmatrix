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
	app.Version = "0.0.1"
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode braille symbols."
	app.Flags = []cli.Flag{
		cli.Float64Flag{
			Name:  "luminosity,l",
			Usage: "(Decimal) Percentage value, between 0 and 1, of luminosity. Defaults to 0.5.",
		},
		cli.BoolFlag{
			Name:  "invert,i",
			Usage: "(Boolean) Inverts colors. Defaults to no inversion.",
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

		var opts []func(enc *dotmatrix.ImageEncoder)
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
