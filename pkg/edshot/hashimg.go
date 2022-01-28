package edshot

import (
	"hash/crc32"
	"image"
)

// HashImage computes a hash value for an image based on its RGBA values.
func HashImage(img image.Image, bounds image.Rectangle) uint32 {
	hash := crc32.NewIEEE()
	row := make([]byte, (bounds.Max.X-bounds.Min.X)*4 / PIXEL_SCALE)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += PIXEL_SCALE {
		i := 0
		for x := bounds.Min.X; x < bounds.Max.X; x += PIXEL_SCALE {
			r, g, b, a := img.At(x, y).RGBA()
			row[i] = uint8(r >> 8)
			i++
			row[i] = uint8(g >> 8)
			i++
			row[i] = uint8(b >> 8)
			i++
			row[i] = uint8(a >> 8)
			i++
		}
		hash.Write(row)
	}
	return hash.Sum32()
}
