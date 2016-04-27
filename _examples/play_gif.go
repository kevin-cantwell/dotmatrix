package main

import (
	"image/gif"
	"os"

	"github.com/kevin-cantwell/dotmatrix"
)

func main() {
	giff, err := gif.DecodeAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	enc := dotmatrix.NewGIFEncoder(dotmatrix.Config{Luminosity: 0.45, Inverted: false})
	if err := enc.Encode(os.Stdout, giff); err != nil {
		panic(err)
	}
}
