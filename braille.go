package dotmatrix

import (
	"bytes"
	"fmt"
	"image/gif"
	"io"
	"time"
)

// Represents an 8 dot braille pattern using x,y coordinates. Eg:
// +----------+
// |(0,0)(1,0)|
// |(0,1)(1,1)|
// |(0,2)(1,2)|
// |(0,3)(1,3)|
// +----------+
type braille [2][4]int

// codePoint maps each point in braille to a dot identifier and
// calculates the corresponding unicode symbol.
// +------+
// |(1)(4)|
// |(2)(5)|
// |(3)(6)|
// |(7)(8)|
// +------+
// See https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering)
func (b braille) codePoint() rune {
	lowEndian := [8]int{b[0][0], b[0][1], b[0][2], b[1][0], b[1][1], b[1][2], b[0][3], b[1][3]}
	var v int
	for i, x := range lowEndian {
		v += int(x) << uint(i)
	}
	return rune(v) + '\u2800'
}

func (b braille) String() string {
	return string(b.codePoint())
}

type GIFEncoder struct {
	imageEncoder *ImageEncoder
}

func NewGIFEncoder(config Config) *GIFEncoder {
	return &GIFEncoder{
		imageEncoder: NewImageEncoder(config),
	}
}

func (enc *GIFEncoder) Encode(w io.Writer, giff *gif.GIF) error {
	var buf bytes.Buffer
	for i := 0; i < len(giff.Image); i++ {
		delay := time.After(time.Duration(giff.Delay[i]) * time.Second / 100)

		err := enc.imageEncoder.Encode(&buf, giff.Image[i])
		if err != nil {
			return err
		}
		var height int
		for {
			c, err := buf.ReadByte()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if c == '\n' {
				height++
			}
			w.Write([]byte{c})
		}
		if i < len(giff.Image)-1 {
			w.Write([]byte("\033[999D"))                     // Move the cursor to the beginning of the line
			w.Write([]byte(fmt.Sprintf("\033[%dA", height))) // Move the cursor to the top of the image
		}

		<-delay
	}
	return nil
}
