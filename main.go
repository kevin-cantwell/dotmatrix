package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
)

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
	jpg, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	dx := jpg.Bounds().Dx()
	dy := jpg.Bounds().Dy()
	var dots [][]int
	for x := 0; x < dx; x++ {
		var col []int
		dots = append(dots, col)
		for y := 0; y < dy; y++ {
			r, _, _, _ := jpg.At(x, y).RGBA()
			if r > 0 {
				col = append(col, 1)
			} else {
				col = append(col, 0)
			}
		}
	}

	braille := Braille{}
	c := braille.Char([2][4]int{
		{1, 1, 1, 1},
		{0, 0, 0, 0},
	})
	fmt.Println(string(c))
}

type Braille struct{}

func (b *Braille) Char(dots [2][4]int) rune {
	// See https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering
	lowEndian := [8]int{dots[0][0], dots[0][1], dots[0][2], dots[1][0], dots[1][1], dots[1][2], dots[0][3], dots[1][3]}
	var v int
	for i, x := range lowEndian {
		v += x << uint(i)
	}
	return rune(v) + '\u2800'
}
