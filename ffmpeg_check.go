package dotmatrix

/*
#cgo pkg-config: libavcodec libavformat libavutil

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>

// Require FFmpeg 8.x (libavcodec >= 61, libavformat >= 61, libavutil >= 59)
// These version numbers correspond to FFmpeg n8.0
#if LIBAVCODEC_VERSION_MAJOR < 61
  #error "FFmpeg 8.0 or later is required. Found libavcodec version too old. Please install FFmpeg 8.x: https://ffmpeg.org/download.html"
#endif

#if LIBAVFORMAT_VERSION_MAJOR < 61
  #error "FFmpeg 8.0 or later is required. Found libavformat version too old. Please install FFmpeg 8.x: https://ffmpeg.org/download.html"
#endif

#if LIBAVUTIL_VERSION_MAJOR < 59
  #error "FFmpeg 8.0 or later is required. Found libavutil version too old. Please install FFmpeg 8.x: https://ffmpeg.org/download.html"
#endif
*/
import "C"
