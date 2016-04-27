package main

import (
	"image/png"
	"os"

	"github.com/kevin-cantwell/dotmatrix"
)

func main() {
	img, err := dotmatrix.Decode(os.Stdin)
	if err != nil {
		panic(err)
	}
	if err := png.Encode(os.Stdout, img); err != nil {
		panic(err)
	}
}
