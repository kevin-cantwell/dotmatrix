package main

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	_ "golang.org/x/image/bmp"

	"github.com/codegangsta/cli"
	"github.com/kevin-cantwell/dotmatrix"
	"golang.org/x/net/context"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "dotmatrix"
	app.Usage = "A command-line tool for encoding images as unicode braille symbols."
	app.UsageText = "1) dotmatrix [options] [file|url]\n" +
		/*      */ "   2) dotmatrix [options] < [file]"
	app.Author = "Kevin Cantwell"
	app.Email = "kevin.cantwell@gmail.com"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "invert,i",
			Usage: "Inverts black and white pixels.",
		},
		cli.StringFlag{
			Name:  "fit,f",
			Usage: "`W,H` = 80,25 scales down the image to fit a terminal size of 80 by 25.",
			Value: func() string {
				w, h, _ := getTerminalSize()
				return fmt.Sprintf("%d,%d", w, h)
			}(),
		},
		cli.Float64Flag{
			Name:  "gamma,g",
			Usage: "GAMMA less than 0 darkens the image and GAMMA greater than 0 lightens it.",
		},
		cli.Float64Flag{
			Name:  "brightness,b",
			Usage: "BRIGHTNESS = -100 gives solid black image. BRIGHTNESS = 100 gives solid white image.",
			Value: 0.0,
		},
		cli.Float64Flag{
			Name:  "contrast,c",
			Usage: "CONTRAST = -100 gives solid grey image. CONTRAST = 100 gives maximum contrast.",
			Value: 0.0,
		},
		cli.Float64Flag{
			Name:  "sharpen,s",
			Usage: "SHARPEN greater than 0 sharpens the image.",
			Value: 0.0,
		},
		cli.BoolFlag{
			Name:  "partymode,p",
			Usage: "Animates gifs in party mode.",
		},
		cli.BoolFlag{
			Name:  "camera,cam",
			Usage: "Use FaceTime camera input (Requires ffmpeg+avfoundation).",
		},
		cli.BoolFlag{
			Name:  "video,vid",
			Usage: "Use video input (Requires ffmpeg).",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("camera") {
			cmd := exec.Command("ffmpeg", "-r", "30", "-f", "avfoundation", "-i", "FaceTime", "-f", "mjpeg", "-loglevel", "panic", "pipe:")
			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			defer stdoutPipe.Close()
			if err := cmd.Start(); err != nil {
				return err
			}
			go func() {
				if err := cmd.Wait(); err != nil {
					exit(err.Error(), 1)
				}
			}()
			return dotmatrix.PlayMJPEG(os.Stdout, stdoutPipe, 30)
		}

		var reader io.Reader

		// Try to parse the args, if there are any, as a file or url
		if input := c.Args().First(); input != "" {
			// Is it a file?
			if file, err := os.Open(input); err == nil {
				reader = file
			} else {
				// Is it a url?
				resp, err := http.Get(input)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				reader = resp.Body
			}
		} else {
			reader = os.Stdin
		}

		if c.Bool("video") {
			return dotmatrix.PlayMJPEG(os.Stdout, reader, 30)
		}

		// Encode image as a dotmatrix pattern
		// return encodeImage(c, img)
		term := dotmatrix.Terminal{
			Gamma:      c.Float64("gamma"),
			Brightness: c.Float64("brightness"),
			Contrast:   c.Float64("contrast"),
			Sharpen:    c.Float64("sharpen"),
			Invert:     c.Bool("invert"),
		}

		// Tee out the reads while we attempt to decode the gif
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)

		// First try to play the input as an animated gif
		if giff, err := gif.DecodeAll(tee); err == nil {
			return term.PlayGIF(giff)
		}

		// Assuming the gif decoing failed, copy the remaining bytes into the tee'd buffer
		if _, err := io.Copy(&buf, reader); err != nil {
			return err
		}

		// Finally try to decode the image as regular image
		img, _, err := image.Decode(&buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		return term.DrawImage(img)
	}

	if err := app.Run(os.Args); err != nil {
		exit(err.Error(), 1)
	}
}

type partyWriter struct {
	ctx      context.Context
	writer   io.Writer
	colors   []int
	colorIdx int
}

var partyLights = []int{
	425,
	227,
	47,
	5, // Blue
	275,
	383,
	419,
	202,
	204,
}

func (w *partyWriter) Write(b []byte) (int, error) {
	if string(b) == "\033[0m" {
		if w.colorIdx >= len(w.colors) {
			w.colorIdx = 0
		}
		n, err := w.writer.Write([]byte(fmt.Sprintf("\033[38;5;%dm", w.colors[w.colorIdx])))
		w.colorIdx++
		select {
		case <-w.ctx.Done():
			w.writer.Write([]byte("\033[0m"))
		default:
		}
		return n, err
	}
	return w.writer.Write(b)
}

func exit(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

func getTerminalSize() (width, height int, err error) {
	var dimensions [4]uint16
	_, _, e := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stderr), // TODO: Figure out why we get "inappropriate ioctl for device" errors if we use stdin or stdout
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&dimensions)),
		0, 0, 0,
	)
	if e != 0 {
		return -1, -1, e
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}
