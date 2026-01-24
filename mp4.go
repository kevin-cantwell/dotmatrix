package dotmatrix

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asticode/go-astiav"
)

// subtitle holds a decoded subtitle with timing information.
type subtitle struct {
	startPTS int64
	endPTS   int64
	text     string
}

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

	// Find video and subtitle streams
	var videoStream *astiav.Stream
	var videoStreamIdx int
	var subtitleStream *astiav.Stream
	var subtitleStreamIdx int
	for _, s := range formatCtx.Streams() {
		switch s.CodecParameters().MediaType() {
		case astiav.MediaTypeVideo:
			if videoStream == nil {
				videoStream = s
				videoStreamIdx = s.Index()
			}
		case astiav.MediaTypeSubtitle:
			if subtitleStream == nil {
				subtitleStream = s
				subtitleStreamIdx = s.Index()
			}
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

	// Set up subtitle storage if subtitle stream exists
	var subtitles []subtitle
	var subtitleTimeBase astiav.Rational
	if subtitleStream != nil {
		subtitleTimeBase = subtitleStream.TimeBase()
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

		// Handle subtitle packets
		if subtitleStream != nil && pkt.StreamIndex() == subtitleStreamIdx {
			// Parse mov_text subtitle format (2-byte length prefix + UTF-8 text)
			data := pkt.Data()
			if len(data) >= 2 {
				textLen := int(binary.BigEndian.Uint16(data[:2]))
				if textLen > 0 && len(data) >= 2+textLen {
					text := strings.TrimSpace(string(data[2 : 2+textLen]))
					if text != "" {
						// Convert subtitle PTS to video stream time base
						pktPTS := pkt.Pts()
						startPTS := pktPTS * int64(subtitleTimeBase.Num()) * int64(timeBase.Den()) / (int64(subtitleTimeBase.Den()) * int64(timeBase.Num()))
						// Duration from packet, convert to video time base
						pktDuration := pkt.Duration()
						durationPTS := pktDuration * int64(subtitleTimeBase.Num()) * int64(timeBase.Den()) / (int64(subtitleTimeBase.Den()) * int64(timeBase.Num()))
						endPTS := startPTS + durationPTS

						subtitles = append(subtitles, subtitle{
							startPTS: startPTS,
							endPTS:   endPTS,
							text:     text,
						})
					}
				}
			}
			pkt.Unref()
			continue
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

			// Overlay subtitle on the last row of the braille output
			imageWidthInChars := filteredImg.Bounds().Dx() / 2
			currentPTS := frame.Pts()
			activeSubtitle := findActiveSubtitle(subtitles, currentPTS)
			if activeSubtitle != "" {
				centered := centerText(activeSubtitle, imageWidthInChars)
				// Move cursor up 1 line, to beginning, write subtitle, then back down
				fmt.Fprintf(p.w, "\033[1A\r%s\033[1B\r", centered)
			}

			p.c.Reset(p.w, rows)
			frame.Unref()
		}
	}
}

// findActiveSubtitle returns the subtitle text that should be displayed at the given PTS.
func findActiveSubtitle(subtitles []subtitle, pts int64) string {
	for _, s := range subtitles {
		if pts >= s.startPTS && pts < s.endPTS {
			return s.text
		}
	}
	return ""
}

// centerText centers text within the given width, truncating if necessary.
func centerText(text string, width int) string {
	text = strings.TrimSpace(text)

	textLen := len([]rune(text))
	if textLen >= width {
		// Truncate to fit
		runes := []rune(text)
		if width > 3 {
			return string(runes[:width-3]) + "..."
		}
		return string(runes[:width])
	}

	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}
