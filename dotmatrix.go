package dotmatrix

import (
	"image"
	"io"
)

type dot int

const (
	filled dot = 1
	nofill dot = 0
)

// WithLuminosity is a functional option for NewImageEncoder that sets the
// luminosity percentage. The value for lum should be between 0 and 1.
//
// For more about functional options, see:
// http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
func WithLuminosity(lum float32) func(enc *ImageEncoder) {
	return func(enc *ImageEncoder) {
		enc.luminosity = lum
	}
}

// WithInvertedColors is a functional option for NewImageEncoder that causes
// colors to be inverted.
//
// For more about functional options, see:
// http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
func WithInvertedColors() func(enc *ImageEncoder) {
	return func(enc *ImageEncoder) {
		enc.invert = true
	}
}

// ImageEncoder encodes an image as a series of braille and line feed (newline)
// unicode characters. Braille symbols are useful for representing monochrome images
// because any rectangle of 2 by 8 pixels can be represented by one of unicode's
// 256 braille symbols:
//   ⠁⠂⠃⠄⠅⠆⠇⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⠒⠓⠔⠕⠖⠗⠘⠙⠚⠛⠜⠝⠞⠟⠠⠡⠢⠣⠤⠥⠦⠧⠨⠩⠪⠫⠬⠭⠮⠯⠰⠱⠲⠳⠴⠵⠶⠷⠸⠹⠺⠻⠼⠽⠾⠿⡀⡁⡂⡃⡄⡅⡆⡇⡈⡉⡊⡋⡌⡍⡎⡏⡐⡑⡒⡓⡔⡕⡖⡗⡘⡙⡚⡛⡜⡝⡞⡟⡠⡡⡢⡣⡤⡥⡦⡧⡨⡩⡪⡫⡬⡭⡮⡯⡰⡱⡲⡳⡴⡵⡶⡷⡸⡹⡺⡻⡼⡽⡾⡿⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾⣿
// ImageEncoder is not safe for concurrent use.
//
// See: https://en.wikipedia.org/wiki/Braille_Patterns
type ImageEncoder struct {
	// writer to which we write the braille representation of the image.
	writer     io.Writer // Output
	luminosity float32   // Percentage
	invert     bool      // Invert colors
}

// NewImageEncoder configures and returns an ImageEncoder. If no options are passed, a default
// luminosity of 50% is used and colors remain un-inverted.
//
// For more about functional options, see:
// http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
func NewImageEncoder(w io.Writer, opts ...func(enc *ImageEncoder)) *ImageEncoder {
	enc := ImageEncoder{
		writer:     w,
		luminosity: 0.5,
		invert:     false,
	}
	for _, opt := range opts {
		opt(&enc)
	}
	return &enc
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
func (enc *ImageEncoder) Encode(img image.Image) error {
	bounds := img.Bounds()

	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X. Looping over Y first and X second is more
	// likely to result in better memory access patterns than X first and Y second.
	for py := bounds.Min.Y; py < bounds.Max.Y; py += 4 {
		for px := bounds.Min.X; px < bounds.Max.X; px += 2 {
			var dots pattern
			// Draw left-right, top-bottom.
			for y := 0; y < 4; y++ {
				for x := 0; x < 2; x++ {
					// Braille symbols are 2x8, which may end up adding
					// pixels to the right or bottom of the image. In those
					// cases we just don't fill the dots.
					if px+x >= bounds.Max.X || py+y >= bounds.Max.Y {
						dots[x][y] = nofill
						continue
					}
					dots[x][y] = enc.dotAt(img, px+x, py+y)
				}
			}
			if _, err := enc.writer.Write([]byte(dots.String())); err != nil {
				return err
			}
		}
		if _, err := enc.writer.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func (enc *ImageEncoder) dotAt(img image.Image, x, y int) dot {
	gray := grayscale(img.At(x, y).RGBA())
	if gray <= float32(0xffff)*(1.0-enc.luminosity) {
		if enc.invert {
			return nofill
		}
		return filled
	}

	if enc.invert {
		return filled
	}
	return nofill
}

// Standard-ish algorithm for determining the best grayscale for human eyes
// 0.21 R + 0.72 G + 0.07 B
func grayscale(r, g, b, a uint32) float32 {
	return 0.21*float32(r) + 0.72*float32(g) + 0.07*float32(b)
}

// Represents an 8 dot braille pattern using x,y coordinates. Eg:
// +----------+
// |(0,0)(1,0)|
// |(0,1)(1,1)|
// |(0,2)(1,2)|
// |(0,3)(1,3)|
// +----------+
type pattern [2][4]dot

// CodePoint maps each point in pattern to a braille number and
// calculates the corresponding unicode symbol.
// +------+
// |(1)(4)|
// |(2)(5)|
// |(3)(6)|
// |(7)(8)|
// +------+
// See https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering)
func (dots pattern) CodePoint() rune {
	lowEndian := [8]dot{dots[0][0], dots[0][1], dots[0][2], dots[1][0], dots[1][1], dots[1][2], dots[0][3], dots[1][3]}
	var v int
	for i, x := range lowEndian {
		v += int(x) << uint(i)
	}
	return rune(v) + '\u2800'
}

func (dots pattern) String() string {
	return string(dots.CodePoint())
}
