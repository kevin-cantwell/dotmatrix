package main

import (
	"image/gif"
	"os"

	"github.com/kevin-cantwell/dotmatrix"
)

func main() {
	dec := dotmatrix.NewGIFDecoder(dotmatrix.Config{Luminosity: 0.8})
	giff, err := dec.Decode(os.Stdin)
	if err != nil {
		panic(err)
	}
	if err := gif.EncodeAll(os.Stdout, giff); err != nil {
		panic(err)
	}
}
