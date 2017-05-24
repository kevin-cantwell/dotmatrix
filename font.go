package dotmatrix

// import (
// 	"image"
// 	"image/draw"
// 	"io/ioutil"
// 	"os"
// 	"strings"

// 	"github.com/golang/freetype"

// 	"golang.org/x/image/font"
// 	"golang.org/x/image/font/basicfont"
// 	"golang.org/x/image/math/fixed"
// )

// func Print(label string) {
// 	// x, y, _ := getTerminalSize()
// 	// bounds := image.Rectangle{
// 	// 	Min: image.Pt(0, 0),
// 	// 	Max: image.Pt(x*2, y*4),
// 	// }
// 	// // panic(bounds)
// 	// paletted := image.NewPaletted(bounds, simplePalette)
// 	// addLabel(paletted, 1, 1, label)

// 	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
// 	addLabel(img, 0, 1*13, "IMPEACH")
// 	// addLabel(img, 0, 2*13, "OGE")
// 	NewEncoder(os.Stdout, draw.Src).Encode(img)
// }

// func PrintText(label string) error {
// 	size := float64(74)
// 	dpi := float64(72)
// 	spacing := float64(1)
// 	hinting := font.HintingNone
// 	fg := image.Black
// 	bg := image.Transparent

// 	fontBytes, err := ioutil.ReadFile("/Library/Fonts/Courier New Bold.ttf")
// 	if err != nil {
// 		return err
// 	}
// 	f, err := freetype.ParseFont(fontBytes)
// 	if err != nil {
// 		return err
// 	}

// 	x, y, _ := getTerminalSize()
// 	rect := image.Rectangle{
// 		Min: image.ZP,
// 		Max: image.Pt(x, y),
// 	}

// 	rgba := image.NewRGBA(rect)
// 	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)

// 	c := freetype.NewContext()
// 	c.SetDPI(dpi)
// 	c.SetFont(f)
// 	c.SetFontSize(size)
// 	c.SetClip(rgba.Bounds())
// 	c.SetDst(rgba)
// 	c.SetSrc(fg)
// 	c.SetHinting(hinting)

// 	pt := freetype.Pt(0, int(c.PointToFixed(size)>>6))
// 	for _, s := range strings.Split(label, "\n") {
// 		_, err = c.DrawString(s, pt)
// 		if err != nil {
// 			return err
// 		}
// 		pt.Y += c.PointToFixed(size * spacing)
// 	}

// 	return NewEncoder(os.Stdout, draw.Src).Encode(rgba)
// }

// func addLabel(img *image.RGBA, x, y int, label string) {
// 	// col := color.RGBA{200, 100, 0, 255}
// 	// point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}
// 	point := fixed.P(x, y)

// 	d := &font.Drawer{
// 		Dst:  img,
// 		Src:  image.Black,        // image.NewUniform(col),
// 		Face: basicfont.Face7x13, //inconsolata.Regular8x16,
// 		Dot:  point,
// 	}
// 	d.DrawString(label)
// }
