package dotmatrix

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
)

// Flushes an image to the io.Writer. E.g. by using braille characters.
type Flusher interface {
	Flush(w io.Writer, img image.Image) error
}

// Filter may alter an image in any way, including resizing it.
// It is applied prior to drawing the image in the dotmatrix palette.
type Filter interface {
	Filter(image.Image) image.Image
}

type noop struct{}

func (noop) Filter(img image.Image) image.Image {
	return img
}

type Config struct {
	Filter  Filter
	Flusher Flusher
	Drawer  draw.Drawer
	// Reset is invoked between animated frames of an image. It can be used to
	// apply custom cursor positioning.
	Reset func(w io.Writer, rows int)
}

var defaultConfig = Config{
	Filter:  noop{},
	Flusher: BrailleFlusher{},
	Drawer:  draw.FloydSteinberg,
}

func mergeConfig(c *Config) Config {
	if c == nil {
		return defaultConfig
	}
	if c.Filter == nil {
		c.Filter = defaultConfig.Filter
	}
	if c.Drawer == nil {
		c.Drawer = defaultConfig.Drawer
	}
	if c.Flusher == nil {
		c.Flusher = defaultConfig.Flusher
	}
	if c.Reset == nil {
		c.Reset = func(w io.Writer, rows int) {
			fmt.Fprintf(w, "\033[999D\033[%dA", rows)
		}
	}
	return *c
}

var defaultPalette = []color.Color{color.Black, color.White, color.Transparent}

type Printer struct {
	w io.Writer
	c Config
}

func Print(w io.Writer, img image.Image) error {
	return NewPrinter(w, &defaultConfig).Print(img)
}

// NewPrinter provides an Printer. If drawer is nil, draw.FloydSteinberg is used.
func NewPrinter(w io.Writer, c *Config) *Printer {
	return &Printer{
		w: w,
		c: mergeConfig(c),
	}
}

/*
Print prints the image as a series of braille and line feed characters and writes
to w. Braille symbols are useful for representing monochrome images
because any 2x4 pixel area can be represented by one of unicode's
256 braille symbols. See: https://en.wikipedia.org/wiki/Braille_Patterns

Each pixel of the image is converted to either black or white by redrawing the
image using the printer's drawer (Floyd Steinberg diffusion, by default) and a
3-color palette of black, white, and transparent. Finally, each 2x4 pixel block
is printed as a braille symbol.

As an example, this output was printed from a 134px by 108px image of Saturn:
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡿⡻⡫⡫⡣⣣⢣⢇⢧⢫⢻⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡟⡟⣝⣜⠼⠼⢚⢚⢚⠓⠷⣧⣇⠧⡳⡱⣻⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡟⣏⡧⠧⠓⠍⡂⡂⠅⠌⠄⠄⠄⡁⠢⡈⣷⡹⡸⣪⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢿⠿⢿⢿⢿⢟⢏⡧⠗⡙⡐⡐⣌⢬⣒⣖⣼⣼⣸⢸⢐⢁⠂⡐⢰⡏⣎⢮⣾⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣽⣾⣶⣿⢿⢻⡱⢕⠋⢅⠢⠱⢼⣾⣾⣿⣿⣿⣿⣿⣿⣿⡇⡇⠢⢁⢂⡯⡪⣪⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⠏⢎⠪⠨⡐⠔⠁⠁⠀⠀⠀⠙⢿⣿⣿⣿⣿⣿⣿⣿⢱⠡⡁⣢⢏⢮⣾⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⢍⢆⢃⢑⠤⠑⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣿⣿⣿⣿⡿⡱⢑⢐⢼⢱⣵⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢿⢫⡱⢊⢂⢢⠢⡃⠌⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⢟⢑⢌⢦⢫⣪⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡻⡱⡑⢅⢢⣢⣳⢱⢑⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠹⡑⡑⡴⡹⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⢝⠜⠨⡐⣴⣵⣿⣗⡧⡣⠢⢈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣜⢎⣷⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡫⡱⠑⡁⣌⣮⣾⣿⣿⣿⣟⡮⡪⡪⡐⠠⠀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⢏⠜⠌⠄⣕⣼⣿⣿⣿⣿⣿⣿⣯⡯⣎⢖⠌⠌⠄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢨⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⢕⠕⢁⠡⣸⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⡽⡮⡪⡪⠨⡂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⢕⠕⢁⢐⢔⣽⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢽⡱⡱⡑⡠⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢟⢕⠕⢁⢐⢰⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⣞⢜⠔⢄⠡⠀⠀⠀⠀⠀⠀⠀⠀⠀⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⡿⡹⡰⠃⢈⠠⣢⣿⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡮⣇⢏⢂⠢⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⢫⢒⡜⠐⠀⢢⣱⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣳⢕⢕⠌⠄⡀⠀⠀⢀⣤⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⡿⡑⣅⠗⠀⡀⣥⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⢙⠙⠿⣿⣿⣿⣿⣿⣿⣿⣿⣯⢮⡪⣂⣢⣬⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⡟⡜⢌⡞⡀⣡⣾⣿⣿⣿⣿⣿⣿⣿⡿⠛⠉⢀⡠⠔⢜⣱⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⡿⡸⡘⢜⣧⣾⣿⣿⣿⣿⣿⣿⠿⢛⡡⠤⡒⢪⣑⣬⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⡇⡇⡣⣷⣿⣿⣿⣿⣿⠿⡛⡣⡋⣕⣬⣶⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣮⣺⣿⣿⣟⣻⣩⣢⣵⣾⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
	⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿
*/
func (p *Printer) Print(img image.Image) error {
	img = redraw(img, p.c.Filter, p.c.Drawer)
	return flush(p.w, img, p.c.Flusher)
}

func redraw(img image.Image, filter Filter, drawer draw.Drawer) *image.Paletted {
	origBounds := img.Bounds()

	img = filter.Filter(img)

	newBounds := img.Bounds()

	scaleX := float64(newBounds.Dx()) / float64(origBounds.Dx())
	scaleY := float64(newBounds.Dy()) / float64(origBounds.Dy())

	// The offset is important because not all images have bounds starting at (0, 0), and
	// the filter may accidentally zero the min bounding point.
	offset := image.Pt(int(float64(origBounds.Min.X)*scaleX), int(float64(origBounds.Min.Y)*scaleY))

	// Create a new paletted image using a monochrome+transparent color palette.
	paletted := image.NewPaletted(img.Bounds(), defaultPalette)
	paletted.Rect = paletted.Bounds().Add(offset)
	drawer.Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)
	return paletted
}

func flush(w io.Writer, img image.Image, flusher Flusher) error {
	return flusher.Flush(w, img)

}
