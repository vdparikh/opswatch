package framehash

import (
	"image"
	"image/color"
	"testing"
)

func TestDistance(t *testing.T) {
	if got := Distance(Hash(0), Hash(3)); got != 2 {
		t.Fatalf("expected distance 2, got %d", got)
	}
}

func TestImageHashChanges(t *testing.T) {
	leftDark := image.NewRGBA(image.Rect(0, 0, 80, 80))
	leftLight := image.NewRGBA(image.Rect(0, 0, 80, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			if x < 40 {
				leftDark.Set(x, y, color.Black)
				leftLight.Set(x, y, color.White)
			} else {
				leftDark.Set(x, y, color.White)
				leftLight.Set(x, y, color.Black)
			}
		}
	}

	if got := Distance(Image(leftDark), Image(leftLight)); got == 0 {
		t.Fatal("expected different hashes")
	}
}
