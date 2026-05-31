// Package gifenc assembles a sequence of encoded image frames (JPEG/PNG) into
// an animated GIF. Pure stdlib — no external dependency. The orchestrator
// composes it with the chromedp screencast primitive to implement `record`.
package gifenc

import (
	"bytes"
	"errors"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // register the JPEG decoder for image.Decode
	_ "image/png"  // register the PNG decoder for image.Decode
)

// Encode assembles `frames` (raw encoded images) into an animated GIF.
// `delays` are per-frame in 1/100s units; entries that are missing or <= 0
// fall back to `defaultDelay`. Frames that fail to decode are skipped.
func Encode(frames [][]byte, delays []int, defaultDelay int) ([]byte, error) {
	if len(frames) == 0 {
		return nil, errors.New("gifenc: no frames")
	}
	if defaultDelay <= 0 {
		defaultDelay = 20
	}
	g := &gif.GIF{LoopCount: 0}
	for i, raw := range frames {
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})
		d := defaultDelay
		if i < len(delays) && delays[i] > 0 {
			d = delays[i]
		}
		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, d)
	}
	if len(g.Image) == 0 {
		return nil, errors.New("gifenc: no decodable frames")
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
