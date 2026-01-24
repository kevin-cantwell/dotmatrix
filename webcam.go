package dotmatrix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asticode/go-astiav"
)

// WebcamPrinter prints webcam frames as braille characters.
type WebcamPrinter struct {
	w io.Writer
	c Config
}

// NewWebcamPrinter creates a new WebcamPrinter.
func NewWebcamPrinter(w io.Writer, c *Config) *WebcamPrinter {
	return &WebcamPrinter{
		w: w,
		c: mergeConfig(c),
	}
}

// Print captures frames from the webcam and renders them as braille.
// If fps is less than zero, frames are rendered as fast as they arrive.
// Otherwise, fps dictates how many frames per second are printed.
func (p *WebcamPrinter) Print(ctx context.Context, fps int) error {
	// Register all devices to enable AVFoundation
	astiav.RegisterAllDevices()

	// Find AVFoundation input format (macOS)
	inputFormat := astiav.FindInputFormat("avfoundation")
	if inputFormat == nil {
		return fmt.Errorf("webcam: avfoundation input format not found (macOS only)")
	}

	// Set up options
	options := astiav.NewDictionary()
	defer options.Free()

	// Set framerate - default to 30 if not specified (AVFoundation requires exact fps)
	if fps <= 0 {
		fps = 30
	}
	options.Set("framerate", fmt.Sprintf("%d", fps), 0)

	// Set pixel format to one supported by most cameras
	options.Set("pixel_format", "uyvy422", 0)

	// Increase probesize to give AVFoundation time to initialize
	options.Set("probesize", "32000000", 0)

	// Allocate format context
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		return fmt.Errorf("webcam: failed to allocate format context")
	}
	defer formatCtx.Free()

	// Open webcam device (device "0" is the first video device)
	if err := formatCtx.OpenInput("0", inputFormat, options); err != nil {
		// Provide helpful error messages for common issues
		errStr := err.Error()
		if errStr == "Operation not permitted" || errStr == "Permission denied" {
			return fmt.Errorf("webcam: permission denied - check System Preferences > Privacy & Security > Camera")
		}
		if errStr == "Device or resource busy" {
			return fmt.Errorf("webcam: camera is in use by another application")
		}
		return fmt.Errorf("webcam: failed to open camera: %w", err)
	}
	defer formatCtx.CloseInput()

	// Find stream info
	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return fmt.Errorf("webcam: finding stream info failed: %w", err)
	}

	// Find video stream
	var videoStream *astiav.Stream
	var videoStreamIdx int
	for _, s := range formatCtx.Streams() {
		if s.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			videoStream = s
			videoStreamIdx = s.Index()
			break
		}
	}
	if videoStream == nil {
		return fmt.Errorf("webcam: no video stream found")
	}

	// Find decoder
	codec := astiav.FindDecoder(videoStream.CodecParameters().CodecID())
	if codec == nil {
		return fmt.Errorf("webcam: decoder not found for codec %s", videoStream.CodecParameters().CodecID())
	}

	// Allocate codec context
	codecCtx := astiav.AllocCodecContext(codec)
	if codecCtx == nil {
		return fmt.Errorf("webcam: failed to allocate codec context")
	}
	defer codecCtx.Free()

	// Copy codec parameters
	if err := videoStream.CodecParameters().ToCodecContext(codecCtx); err != nil {
		return fmt.Errorf("webcam: copying codec parameters failed: %w", err)
	}

	// Open codec
	if err := codecCtx.Open(codec, nil); err != nil {
		return fmt.Errorf("webcam: opening codec failed: %w", err)
	}

	// Allocate packet
	pkt := astiav.AllocPacket()
	if pkt == nil {
		return fmt.Errorf("webcam: failed to allocate packet")
	}
	defer pkt.Free()

	// Allocate frame
	frame := astiav.AllocFrame()
	if frame == nil {
		return fmt.Errorf("webcam: failed to allocate frame")
	}
	defer frame.Free()

	// Create software scale context for converting to RGBA
	var swsCtx *astiav.SoftwareScaleContext
	var dstFrame *astiav.Frame

	// Calculate frame duration for fps limiting
	var frameDuration time.Duration
	if fps > 0 {
		frameDuration = time.Second / time.Duration(fps)
	}

	var rows int
	var lastFrameTime time.Time

	// Read frames
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read frame
		if err := formatCtx.ReadFrame(pkt); err != nil {
			if errors.Is(err, astiav.ErrEof) {
				return nil
			}
			// EAGAIN means no frame available yet, just retry
			if errors.Is(err, astiav.ErrEagain) {
				continue
			}
			// Also check error string for "Resource temporarily unavailable" (EAGAIN on some systems)
			if strings.Contains(err.Error(), "Resource temporarily unavailable") {
				continue
			}
			return fmt.Errorf("webcam: reading frame failed: %w", err)
		}

		// Skip non-video packets
		if pkt.StreamIndex() != videoStreamIdx {
			pkt.Unref()
			continue
		}

		// Send packet to decoder
		if err := codecCtx.SendPacket(pkt); err != nil {
			pkt.Unref()
			continue
		}
		pkt.Unref()

		// Receive frames from decoder
		for {
			if err := codecCtx.ReceiveFrame(frame); err != nil {
				if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
					break
				}
				return fmt.Errorf("webcam: receiving frame failed: %w", err)
			}

			// Initialize scaler on first frame
			if swsCtx == nil {
				var err error
				swsCtx, err = astiav.CreateSoftwareScaleContext(
					frame.Width(),
					frame.Height(),
					frame.PixelFormat(),
					frame.Width(),
					frame.Height(),
					astiav.PixelFormatRgba,
					astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
				)
				if err != nil {
					frame.Unref()
					return fmt.Errorf("webcam: creating software scale context failed: %w", err)
				}
				defer swsCtx.Free()

				dstFrame = astiav.AllocFrame()
				if dstFrame == nil {
					frame.Unref()
					return fmt.Errorf("webcam: failed to allocate destination frame")
				}
				defer dstFrame.Free()
			}

			// Rate limiting for fps > 0
			if fps > 0 && !lastFrameTime.IsZero() {
				elapsed := time.Since(lastFrameTime)
				if sleepDuration := frameDuration - elapsed; sleepDuration > 0 {
					select {
					case <-ctx.Done():
						frame.Unref()
						return ctx.Err()
					case <-time.After(sleepDuration):
					}
				}
			}
			lastFrameTime = time.Now()

			// Scale frame to RGBA
			if err := swsCtx.ScaleFrame(frame, dstFrame); err != nil {
				frame.Unref()
				return fmt.Errorf("webcam: scaling frame failed: %w", err)
			}

			// Convert to Go image
			img, err := dstFrame.Data().GuessImageFormat()
			if err != nil {
				frame.Unref()
				return fmt.Errorf("webcam: guessing image format failed: %w", err)
			}
			if err := dstFrame.Data().ToImage(img); err != nil {
				frame.Unref()
				return fmt.Errorf("webcam: converting frame to image failed: %w", err)
			}

			// Apply filters and convert to paletted
			filteredImg := redraw(img, p.c.Filter, p.c.Drawer)

			// Flush to output
			if err := flush(p.w, filteredImg, p.c.Flusher); err != nil {
				frame.Unref()
				return err
			}

			// Calculate rows for cursor reset
			if rows == 0 {
				rows = filteredImg.Bounds().Dy() / 4
				if filteredImg.Bounds().Dy()%4 != 0 {
					rows++
				}
			}

			p.c.Reset(p.w, rows)
			frame.Unref()
		}
	}
}
