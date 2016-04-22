package dotmatrix

// import (
// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// )

// // ⡪⣛
// //
// var testCanvas = canvas{
// 	{ // mx1
// 		braille{ // my1
// 			{white, black, white, black}, // x1,y1-4
// 			{black, white, black, white}, // x2,y1-4
// 		},
// 		braille{ // my1
// 			{black, black, black, black},                         // x1,y1-4
// 			{transparent, transparent, transparent, transparent}, // x2,y1-4
// 		},
// 	},
// 	{ // mx2
// 		braille{ // my2
// 			{black, black, white, black}, // x1,y1-4
// 			{black, black, white, black}, // x2,y1-4
// 		},
// 		braille{ // my2
// 			{transparent, transparent, transparent, transparent}, // x1,y1-4
// 			{black, black, black, black},                         // x2,y1-4
// 		},
// 	},
// }

// var _ = Describe("canvas", func() {
// 	Describe("#At", func() {
// 		It("Should return the color at the given coordinate.", func() {

// 			// copy(c, testCanvas)
// 			Expect("\n" + testCanvas.String()).To(Equal("\n⡪⣛\n"))
// 		})
// 	})
// })
