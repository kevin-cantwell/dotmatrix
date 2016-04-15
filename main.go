package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"syscall"
	"unsafe"

	"github.com/nfnt/resize"
)

type dot int

const (
	black dot = 1
	white dot = 0
)

var (
	luminosity = flag.Float64("lum", 50, "the percentage luminosity cutoff to determine black or white pixels")
)

func main() {
	flag.Parse()
	fmt.Println(*luminosity)

	img, _, err := image.Decode(os.Stdin)
	if err != nil {
		panic(err)
	}

	w, _, err := GetTerminalSize()
	if err != nil {
		panic(err)
	}

	w = w * 2 // Since each symbol is two dots wide

	img = resize.Thumbnail(uint(w), uint(img.Bounds().Dy()), img, resize.NearestNeighbor)

	var picture string

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
					dots[x][y] = dotAt(img, px+x, py+y)
				}
			}
			picture += dots.String()
		}
		picture += "\n"
	}

	fmt.Println(picture)
}

func dotAt(img image.Image, x, y int) dot {
	v := grayByLuminosity(img.At(x, y).RGBA())
	// 32,767 is half bright
	if v <= uint32(65535/(100 / *luminosity)) {
		return white
	}

	return black
}

// 0.21 R + 0.72 G + 0.07 B
func grayByLuminosity(r, g, b, a uint32) uint32 {
	weighted := 0.21*float32(r) + 0.72*float32(g) + 0.07*float32(b)
	return uint32(weighted)
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
