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

// Filter is a draw.Drawer that can alter an image via the Filter method.
type Filter interface {
	draw.Drawer
	Filter(image.Image) image.Image
}

type diffuseFilter struct{}

func (diffuseFilter) Filter(img image.Image) image.Image {
	return img
}

func (diffuseFilter) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
	draw.FloydSteinberg.Draw(dst, r, src, sp)
}

type Encoder struct {
	w io.Writer
	f Filter
}

func Encode(w io.Writer, img image.Image) error {
	return NewEncoder(w, nil).Encode(img)
}

// NewEncoder provides an Encoder. If drawer is nil, draw.FloydSteinberg is used.
func NewEncoder(w io.Writer, f Filter) *Encoder {
	if f == nil {
		f = diffuseFilter{}
	}
	return &Encoder{
		w: w,
		f: f,
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
	// Filter and Draw the image
	img = enc.redraw(img)
	// converted := convertToMonochrome(enc.d, img)
	bounds := img.Bounds()

	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X.
	// Looping over Y first and X second is more likely to result in better memory
	// access patterns than X first and Y second.
	for py := bounds.Min.Y; py < bounds.Max.Y; py += 4 {
		for px := bounds.Min.X; px < bounds.Max.X; px += 2 {
			var b Braille
			// Draw left-right, top-bottom.
			for y := 0; y < 4; y++ {
				for x := 0; x < 2; x++ {
					if px+x >= bounds.Max.X || py+y >= bounds.Max.Y {
						continue
					}
					// Always bet on black
					if img.At(px+x, py+y) == color.Black {
						b[x][y] = 1
					}
				}
			}
			if _, err := enc.w.Write([]byte(b.String())); err != nil {
				return err
			}
		}
		if _, err := enc.w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

var defaultPalette = []color.Color{color.Black, color.White, color.Transparent}

func (enc *Encoder) redraw(img image.Image) *image.Paletted {
	img = enc.f.Filter(img)
	// Create a new paletted image using a monochrome+transparent color palette.
	paletted := image.NewPaletted(img.Bounds(), defaultPalette)
	enc.f.Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)
	return paletted
}

func convertToMonochrome(drawer draw.Drawer, img image.Image) *image.Paletted {
	// Create a new paletted image using a monochrome+transparent color palette.
	paletted := image.NewPaletted(img.Bounds(), defaultPalette)
	drawer.Draw(paletted, paletted.Bounds(), img, img.Bounds().Min)
	return paletted
}
