package framehash

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math/bits"
	"os"
)

const sampleSize = 8

type Hash uint64

func File(path string) (Hash, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}
	return Image(img), nil
}

func Image(img image.Image) Hash {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return 0
	}

	var values [sampleSize * sampleSize]uint8
	var total int
	for y := 0; y < sampleSize; y++ {
		py := bounds.Min.Y + y*height/sampleSize + height/(sampleSize*2)
		if py >= bounds.Max.Y {
			py = bounds.Max.Y - 1
		}
		for x := 0; x < sampleSize; x++ {
			px := bounds.Min.X + x*width/sampleSize + width/(sampleSize*2)
			if px >= bounds.Max.X {
				px = bounds.Max.X - 1
			}
			r, g, b, _ := img.At(px, py).RGBA()
			luma := uint8(((299*r + 587*g + 114*b) / 1000) >> 8)
			values[y*sampleSize+x] = luma
			total += int(luma)
		}
	}

	avg := total / len(values)
	var hash uint64
	for i, value := range values {
		if int(value) >= avg {
			hash |= 1 << uint(i)
		}
	}
	return Hash(hash)
}

func Distance(a, b Hash) int {
	return bits.OnesCount64(uint64(a ^ b))
}

func RegisterFormats() {
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("jpg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
}

func (h Hash) String() string {
	return fmt.Sprintf("%016x", uint64(h))
}
