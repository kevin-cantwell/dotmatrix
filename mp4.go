package dotmatrix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/asticode/go-astiav"
)

// MP4Printer prints MP4 video frames as braille characters.
type MP4Printer struct {
	w io.Writer
	c Config
}

// NewMP4Printer creates a new MP4Printer.
func NewMP4Printer(w io.Writer, c *Config) *MP4Printer {
	return &MP4Printer{
		w: w,
		c: mergeConfig(c),
	}
}

// Print plays an MP4 video from a file path. If fps is less than zero, it will
// use the video's native framerate. Otherwise, fps dictates how many frames per
// second are printed.
func (p *MP4Printer) Print(ctx context.Context, inputPath string, fps int) error {
	// Allocate packet
	pkt := astiav.AllocPacket()
	if pkt == nil {
		return fmt.Errorf("mp4: failed to allocate packet")
	}
	defer pkt.Free()

	// Allocate frame
	frame := astiav.AllocFrame()
	if frame == nil {
		return fmt.Errorf("mp4: failed to allocate frame")
	}
	defer frame.Free()

	// Allocate format context
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		return fmt.Errorf("mp4: failed to allocate format context")
	}
	defer formatCtx.Free()

	// Open input
	if err := formatCtx.OpenInput(inputPath, nil, nil); err != nil {
		return fmt.Errorf("mp4: opening input failed: %w", err)
	}
	defer formatCtx.CloseInput()

	// Find stream info
	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return fmt.Errorf("mp4: finding stream info failed: %w", err)
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
		return fmt.Errorf("mp4: no video stream found")
	}

	// Find decoder
	codec := astiav.FindDecoder(videoStream.CodecParameters().CodecID())
	if codec == nil {
		return fmt.Errorf("mp4: decoder not found for codec %s", videoStream.CodecParameters().CodecID())
	}

	// Allocate codec context
	codecCtx := astiav.AllocCodecContext(codec)
	if codecCtx == nil {
		return fmt.Errorf("mp4: failed to allocate codec context")
	}
	defer codecCtx.Free()

	// Copy codec parameters
	if err := videoStream.CodecParameters().ToCodecContext(codecCtx); err != nil {
		return fmt.Errorf("mp4: copying codec parameters failed: %w", err)
	}

	// Open codec
	if err := codecCtx.Open(codec, nil); err != nil {
		return fmt.Errorf("mp4: opening codec failed: %w", err)
	}

	// Create software scale context for converting to RGBA
	var swsCtx *astiav.SoftwareScaleContext
	var dstFrame *astiav.Frame

	// Get stream time base for timing calculations
	timeBase := videoStream.TimeBase()

	// Calculate frame duration for fixed fps mode
	var frameDuration time.Duration
	if fps > 0 {
		frameDuration = time.Second / time.Duration(fps)
	}

	var rows int
	var playbackStart time.Time  // Wall clock time when playback started
	var firstPTS int64           // PTS of the first frame
	var frameCount int64         // Frame counter for fixed fps mode
	var initialized bool

	// Read packets
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
			return fmt.Errorf("mp4: reading frame failed: %w", err)
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
				return fmt.Errorf("mp4: receiving frame failed: %w", err)
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
					return fmt.Errorf("mp4: creating software scale context failed: %w", err)
				}
				defer swsCtx.Free()

				dstFrame = astiav.AllocFrame()
				if dstFrame == nil {
					frame.Unref()
					return fmt.Errorf("mp4: failed to allocate destination frame")
				}
				defer dstFrame.Free()
			}

			// Scale frame to RGBA
			if err := swsCtx.ScaleFrame(frame, dstFrame); err != nil {
				frame.Unref()
				return fmt.Errorf("mp4: scaling frame failed: %w", err)
			}

			// Convert to Go image
			img, err := dstFrame.Data().GuessImageFormat()
			if err != nil {
				frame.Unref()
				return fmt.Errorf("mp4: guessing image format failed: %w", err)
			}
			if err := dstFrame.Data().ToImage(img); err != nil {
				frame.Unref()
				return fmt.Errorf("mp4: converting frame to image failed: %w", err)
			}

			// Initialize timing on first frame
			if !initialized {
				playbackStart = time.Now()
				firstPTS = frame.Pts()
				initialized = true
			}

			// Calculate target display time and wait if needed
			var targetTime time.Time
			if fps > 0 {
				// Fixed framerate: target time based on frame count
				targetTime = playbackStart.Add(time.Duration(frameCount) * frameDuration)
			} else if fps < 0 {
				// Native timing: target time based on PTS
				ptsDiff := frame.Pts() - firstPTS
				videoTime := time.Duration(float64(ptsDiff) * float64(timeBase.Num()) / float64(timeBase.Den()) * float64(time.Second))
				targetTime = playbackStart.Add(videoTime)
			}

			// Sleep until target time (if we're ahead of schedule)
			if sleepDuration := time.Until(targetTime); sleepDuration > 0 {
				select {
				case <-ctx.Done():
					frame.Unref()
					return ctx.Err()
				case <-time.After(sleepDuration):
				}
			}
			frameCount++

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
