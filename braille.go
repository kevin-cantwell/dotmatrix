package dotmatrix

import (
	"image"
	"image/color"
	"io"
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

type BrailleFlusher struct{}

func (BrailleFlusher) Flush(w io.Writer, img image.Image) error {
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
