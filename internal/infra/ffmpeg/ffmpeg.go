// Package ffmpeg is a thin, optional adapter around the `ffmpeg` binary used
// to encode a frame sequence to MP4. If ffmpeg is not installed, Available()
// reports false and callers degrade gracefully (the `record` action falls
// back to a clear error rather than silently producing a GIF).
package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Available reports whether the ffmpeg binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// EncodeMP4 writes `frames` (JPEG-encoded) to a temp directory and invokes
// ffmpeg to assemble an MP4 at `fps` frames/second. The command is bound to
// `ctx` so a task timeout interrupts a hung encode.
func EncodeMP4(ctx context.Context, frames [][]byte, fps int) ([]byte, error) {
	if !Available() {
		return nil, fmt.Errorf("ffmpeg not found on PATH")
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("ffmpeg: no frames to encode")
	}
	if fps <= 0 {
		fps = 5
	}
	dir, err := os.MkdirTemp("", "webtasks-mp4-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	for i, f := range frames {
		name := filepath.Join(dir, fmt.Sprintf("frame%05d.png", i))
		if err := os.WriteFile(name, f, 0o644); err != nil {
			return nil, err
		}
	}
	outPath := filepath.Join(dir, "out.mp4")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-framerate", fmt.Sprint(fps),
		"-i", filepath.Join(dir, "frame%05d.png"),
		"-pix_fmt", "yuv420p",
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, out)
	}
	return os.ReadFile(outPath)
}
