package dotmatrix

import (
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

const (
	black       mono = 1
	white       mono = 0
	transparent mono = -1
)

var (
	DefaultConfig = Config{Luminosity: 0.5, Inverted: false}
)

type Config struct {
	Luminosity float32
	Inverted   bool
}

type Image struct {
	model  colorModel
	bounds image.Rectangle
	pixels []color.Color
	stride int
}

// ColorModel returns the Image's color model.
func (img *Image) ColorModel() color.Model {
	return img.model
}

// Bounds returns the domain for which At can return non-zero color.
// The bounds do not necessarily contain the point (0, 0).
func (img *Image) Bounds() image.Rectangle {
	return img.bounds
}

// At returns the color of the pixel at (x, y).
// At(Bounds().Min.X, Bounds().Min.Y) returns the upper-left pixel of the grid.
// At(Bounds().Max.X-1, Bounds().Max.Y-1) returns the lower-right one.
func (img *Image) At(x, y int) color.Color {
	return img.at(x, y)
}

func (img *Image) at(x, y int) mono {
	x = x - img.bounds.Min.X
	y = y - img.bounds.Min.Y
	i := y*img.stride + x
	if i < len(img.pixels) {
		return img.pixels[i].(mono)
	}
	return transparent
}

type colorModel struct {
	luminosity float32
	inverted   bool
}

func (m colorModel) Convert(c color.Color) color.Color {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return transparent
	}
	gray := 0.21*float32(r) + 0.72*float32(g) + 0.07*float32(b)
	isWhite := float32(0xffff)*(1-m.luminosity) < gray
	if isWhite {
		if !m.inverted {
			return white
		} else {
			return black
		}
	}
	if !m.inverted {
		return black
	} else {
		return white
	}
}

type mono int

func (c mono) RGBA() (r, g, b, a uint32) {
	switch c {
	case black:
		return 0, 0, 0, 0xffff
	case white:
		return 0xffff, 0xffff, 0xffff, 0xffff
	case transparent:
		return 0, 0, 0, 0
	default:
		panic("dotmatrix: unknown color value")
	}
}

// ImageEncoder encodes an image as a series of braille and line feed (newline)
// unicode characters. Braille symbols are useful for representing monochrome images
// because any rectangle of 2 by 8 pixels can be represented by one of unicode's
// 256 braille symbols:
//   ⠁⠂⠃⠄⠅⠆⠇⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⠒⠓⠔⠕⠖⠗⠘⠙⠚⠛⠜⠝⠞⠟
//  ⠠⠡⠢⠣⠤⠥⠦⠧⠨⠩⠪⠫⠬⠭⠮⠯⠰⠱⠲⠳⠴⠵⠶⠷⠸⠹⠺⠻⠼⠽⠾⠿
//  ⡀⡁⡂⡃⡄⡅⡆⡇⡈⡉⡊⡋⡌⡍⡎⡏⡐⡑⡒⡓⡔⡕⡖⡗⡘⡙⡚⡛⡜⡝⡞⡟
//  ⡠⡡⡢⡣⡤⡥⡦⡧⡨⡩⡪⡫⡬⡭⡮⡯⡰⡱⡲⡳⡴⡵⡶⡷⡸⡹⡺⡻⡼⡽⡾⡿
//  ⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟
//  ⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿
//  ⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟
//  ⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾⣿
//
// ImageEncoder is not safe for concurrent use.
//
// See: https://en.wikipedia.org/wiki/Braille_Patterns
type ImageEncoder struct {
	config Config
}

// NewImageEncoder configures and returns an ImageEncoder. If no options are passed, a default
// luminosity of 50% is used and colors remain un-inverted.
//
// For more about functional options, see:
// http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
func NewImageEncoder(config Config) *ImageEncoder {
	return &ImageEncoder{
		config: config,
	}
}

// Encodes the image as a series of braille and line feed characters and writes
// to ImageEncoder's internal writer. Each pixel of the image is converted to
// either black or white by:
//
// 1) Calculating the grayscale value according to the following algorithm:
// 0.21 R + 0.72 G + 0.07 B; then
//
// 2) Choosing black or white by comparing it to the luminosity option (which
// itself defaults to 50% if left unset).
//
// A sample encoding of an 134px by 107px image of Saturn looks like:
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠛⠋⠉⠁⠀⠀⠀⠂⠩⡻⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠋⢉⣀⠤⠤⠒⠒⠒⠒⠲⣶⣄⠀⠀⠁⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠟⢉⡠⠄⠒⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⠀⠀⢢⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠿⠿⠿⠿⠟⢋⡡⠔⠊⠁⠀⠀⣀⣀⣠⣤⣄⣀⠀⠀⠀⠀⠀⢀⡏⠀⢠⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⣾⠿⠛⠁⠄⠊⠁⠀⠀⢤⣴⣾⣿⣿⣿⣿⣿⣿⣿⡇⠀⠀⠀⢀⡞⠀⣠⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣿⣿⣿⣿⡿⠁⠀⠀⢠⠊⢀⣼⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣿⣿⣿⣿⡿⠁⠀⢀⡔⠁⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠋⠠⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⠏⠀⠀⡠⠊⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠛⡡⠐⠁⢀⣠⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⠁⠀⡠⠊⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢉⠔⠈⠀⣠⣶⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡠⠊⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠋⡡⠂⠁⢀⣴⣾⣿⣿⣿⣿⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⢋⠠⠊⠀⣠⣴⣿⣿⣿⣿⣿⣿⣿⣇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢁⠔⠁⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢁⠔⠁⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢁⡔⠁⠀⢀⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⡿⠁⡠⠊⠀⠀⣠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⠋⢀⡜⠁⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⡿⠁⢠⠎⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠋⠛⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⡀⢀⣀⣤⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⡟⠀⠀⡞⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⡿⠟⠉⠀⣀⠤⠒⣁⣤⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⡿⡀⠀⠸⣧⣾⣿⣿⣿⣿⣿⣿⠿⢛⣁⠤⠔⠂⣁⣤⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⡇⠇⢀⣴⣿⣿⣿⣿⣿⠿⠛⠛⠉⣁⣠⣴⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣮⣾⣿⣿⣿⣛⣉⣤⣤⣶⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
//   ⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿⠿
func (enc *ImageEncoder) Encode(w io.Writer, input image.Image) error {
	img := convert(input, enc.config)
	bounds := img.Bounds()

	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X.
	// Looping over Y first and X second is more likely to result in better memory
	// access patterns than X first and Y second.
	for py := bounds.Min.Y; py < bounds.Max.Y; py += 4 {
		for px := bounds.Min.X; px < bounds.Max.X; px += 2 {
			var b braille
			// Draw left-right, top-bottom.
			for y := 0; y < 4; y++ {
				for x := 0; x < 2; x++ {
					// The color model will handle black/white inversion, however
					// transparent pixels need to be drawn as black if inversion is set.
					clr := img.at(px+x, py+y)
					if clr == black || (clr == transparent && enc.config.Inverted) {
						b[x][y] = 1
					}
				}
			}
			if _, err := w.Write([]byte(b.String())); err != nil {
				return err
			}
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

type ImageDecoder struct {
	config Config
}

func NewImageDecoder(config Config) *ImageDecoder {
	return &ImageDecoder{
		config: config,
	}
}

func (dec *ImageDecoder) Decode(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return convert(img, dec.config), nil
}

func Decode(r io.Reader) (image.Image, error) {
	dec := NewImageDecoder(DefaultConfig)
	return dec.Decode(r)
}

func convert(img image.Image, config Config) *Image {
	if casted, ok := img.(*Image); ok {
		return casted
	}

	bounds := img.Bounds()
	converted := Image{
		bounds: bounds,
		pixels: make([]color.Color, bounds.Dx()*bounds.Dy()),
		stride: bounds.Dx(),
		model: colorModel{
			luminosity: config.Luminosity,
			inverted:   config.Inverted,
		},
	}

	var i int

	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X.
	// Looping over Y first and X second is more likely to result in better memory
	// access patterns than X first and Y second.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			converted.pixels[i] = converted.model.Convert(img.At(x, y))
			i++
		}
	}

	return &converted
}
