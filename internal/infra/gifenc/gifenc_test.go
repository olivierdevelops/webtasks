package gifenc

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

// pngFrame builds a tiny solid-colour PNG so Encode has something to decode.
func pngFrame(c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestEncodeAssemblesFrames(t *testing.T) {
	frames := [][]byte{
		pngFrame(color.RGBA{255, 0, 0, 255}),
		pngFrame(color.RGBA{0, 255, 0, 255}),
		pngFrame(color.RGBA{0, 0, 255, 255}),
	}
	out, err := Encode(frames, nil, 20)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("result is not a valid GIF: %v", err)
	}
	if len(g.Image) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(g.Image))
	}
}

func TestEncodeRejectsEmpty(t *testing.T) {
	if _, err := Encode(nil, nil, 20); err == nil {
		t.Fatal("expected an error for zero frames")
	}
}

func TestEncodeSkipsUndecodableFrames(t *testing.T) {
	frames := [][]byte{
		pngFrame(color.RGBA{255, 0, 0, 255}),
		[]byte("not an image"),
	}
	out, err := Encode(frames, nil, 20)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	g, _ := gif.DecodeAll(bytes.NewReader(out))
	if len(g.Image) != 1 {
		t.Fatalf("expected 1 decodable frame, got %d", len(g.Image))
	}
}
