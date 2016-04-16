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

type ImageOpt func(enc *ImageEncoder)

// WithLuminosity sets the luminosity percentage.
func WithLuminosity(lum float32) ImageOpt {
	return func(enc *ImageEncoder) {
		enc.luminosity = lum
	}
}

// If used, colors are inverted.
func WithInvertedColors() ImageOpt {
	return func(enc *ImageEncoder) {
		enc.invert = true
	}
}

type ImageEncoder struct {
	writer     io.Writer // Output
	luminosity float32   // Percentage
	invert     bool      // Invert colors
}

func NewImageEncoder(w io.Writer, opts ...ImageOpt) *ImageEncoder {
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
	if gray <= float32(0xffff)*enc.luminosity {
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
