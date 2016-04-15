package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
)

func main2() {
	var dots pattern
	dots[0] = [4]int{
		0,
		1,
		0,
		1,
	}
	dots[1] = [4]int{
		1,
		0,
		1,
		0,
	}
	fmt.Println(dots.String())
}

func main() {
	// for i := int32(0); i < 16*16; i++ {
	// 	c := '\u2800' + i
	// 	fmt.Printf("%d, %c⣿\n", c, c)
	// }
	// spinner := []rune{'⣾', '⣷', '⣯', '⣟', '⡿', '⢿', '⣻', '⣽'}

	f, err := os.Open("mono.jpg")
	if err != nil {
		panic(err)
	}
	jpg, t, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	fmt.Printf("%T %s\n", jpg, t)

	var picture string

	dx := jpg.Bounds().Dx()
	dy := jpg.Bounds().Dy()

	// Create symbols, left-right & top-bottom.
	for py := 0; py < dy; py += 4 {
		for px := 0; px < dx; px += 2 {
			var dots pattern
			// Draw left-right, top-bottom.
			for y := 0; y < 4; y++ {
				for x := 0; x < 2; x++ {
					dots[x][y] = dotAt(jpg, px+x, py+y)
				}
			}
			picture += dots.String()
		}
		picture += "\n"
	}

	fmt.Println(picture)
}

func dotAt(img image.Image, x, y int) int {
	_, _, _, r := img.At(x, y).RGBA()
	if r > 0 {
		return 1
	}
	return 0
}

// Represents an 8 dot braille pattern using x,y coordinates. Eg:
// +----------+
// |(0,0)(1,0)|
// |(0,1)(1,1)|
// |(0,2)(1,2)|
// |(0,3)(1,3)|
// +----------+
type pattern [2][4]int

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
	lowEndian := [8]int{dots[0][0], dots[0][1], dots[0][2], dots[1][0], dots[1][1], dots[1][2], dots[0][3], dots[1][3]}
	var v int
	for i, x := range lowEndian {
		v += x << uint(i)
	}
	return rune(v) + '\u2800'
}

func (dots pattern) String() string {
	return string(dots.CodePoint())
}
