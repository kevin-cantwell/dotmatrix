package dotmatrix

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/bmp"
)

// Braille epresents an 8 dot braille pattern in x,y coordinates space. Eg:
//   +----------+
//   |(0,0)(1,0)|
//   |(0,1)(1,1)|
//   |(0,2)(1,2)|
//   |(0,3)(1,3)|
//   +----------+
type Braille [2][4]int

// Rune maps each point in braille to a dot identifier and
// calculates the corresponding unicode symbol.
//   +------+
//   |(1)(4)|
//   |(2)(5)|
//   |(3)(6)|
//   |(7)(8)|
//   +------+
// See https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering)
func (b Braille) Rune() rune {
	lowEndian := [8]int{b[0][0], b[0][1], b[0][2], b[1][0], b[1][1], b[1][2], b[0][3], b[1][3]}
	var v int
	for i, x := range lowEndian {
		v += int(x) << uint(i)
	}
	return rune(v) + '\u2800'
}

// String returns a unicode braille character. One of:
//  ⣿ ⠁⠂⠃⠄⠅⠆⠇⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⠒⠓⠔⠕⠖⠗⠘⠙⠚⠛⠜⠝⠞⠟⠠⠡⠢⠣⠤⠥⠦⠧⠨⠩⠪⠫⠬⠭⠮⠯⠰⠱⠲⠳⠴⠵⠶⠷⠸⠹⠺⠻⠼⠽⠾⠿⡀⡁⡂⡃⡄⡅⡆⡇⡈⡉⡊⡋⡌⡍⡎⡏⡐⡑⡒⡓⡔⡕⡖⡗⡘⡙⡚⡛⡜⡝⡞⡟⡠⡡⡢⡣⡤⡥⡦⡧⡨⡩⡪⡫⡬⡭⡮⡯⡰⡱⡲⡳⡴⡵⡶⡷⡸⡹⡺⡻⡼⡽⡾⡿⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾
func (b Braille) String() string {
	return string(b.Rune())
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
	Filter Filter
	Drawer draw.Drawer
}

var defaultConfig = Config{
	Filter: noop{},
	Drawer: draw.FloydSteinberg,
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
	return *c
}

var defaultPalette = []color.Color{color.Black, color.White, color.Transparent}

type Encoder struct {
	w io.Writer
	c Config
}

func Encode(w io.Writer, img image.Image) error {
	return NewEncoder(w, &defaultConfig).Encode(img)
}

// NewEncoder provides an Encoder. If drawer is nil, draw.FloydSteinberg is used.
func NewEncoder(w io.Writer, c *Config) *Encoder {
	return &Encoder{
		w: w,
		c: mergeConfig(c),
	}
}

/*
Encode encodes the image as a series of braille and line feed characters and writes
to w. Braille symbols are useful for representing monochrome images
because any 2x4 pixel area can be represented by one of unicode's
256 braille symbols. See: https://en.wikipedia.org/wiki/Braille_Patterns

Each pixel of the image is converted to either black or white by redrawing the
image using the encoder's drawer (Floyd Steinberg diffusion, by default) and a
3-color palette of black, white, and transparent. Finally, each 2x4 pixel block
is encoded as a braille symbol.

As an example, this output was encoded from a 134px by 108px image of Saturn:
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
func (enc *Encoder) Encode(img image.Image) error {
	img = redraw(img, enc.c.Filter, enc.c.Drawer)
	return flushBraille(enc.w, img)
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

func flushBraille(w io.Writer, img image.Image) error {
	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X.
	// Looping over Y first and X second is more likely to result in better memory
	// access patterns than X first and Y second.
	bounds := img.Bounds()
	for py := bounds.Min.Y; py < bounds.Max.Y; py += 4 {
		for px := bounds.Min.X; px < bounds.Max.X; px += 2 {
			var b Braille
			// Draw left-right, top-bottom.
			for y := 0; y < 4; y++ {
				for x := 0; x < 2; x++ {
					if px+x >= bounds.Max.X || py+y >= bounds.Max.Y {
						continue
					}
					// Always bet on black!
					if img.At(px+x, py+y) == color.Black {
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
