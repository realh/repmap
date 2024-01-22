package repton

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

// ImageCompare returns true if the content of both regions is the same.
// Either or both regions may be nil to compare the entire image. Returns
// true if both images are the same.
func ImagesAreEqualVerbose(img1 image.Image, region1 *image.Rectangle,
	img2 image.Image, region2 *image.Rectangle, verbose bool,
) bool {
	whole1 := false
	if region1 == nil {
		r1 := img1.Bounds()
		region1 = &r1
		whole1 = true
	}
	whole2 := false
	if region2 == nil {
		r2 := img2.Bounds()
		region2 = &r2
		whole2 = true
	}
	if verbose {
		r1desc := fmt.Sprintf("%v", region1)
		if whole1 {
			r1desc += " (whole)"
		} else {
			r1desc += fmt.Sprintf(" (from %v)", img1.Bounds())
		}
		r2desc := fmt.Sprintf("%v", region2)
		if whole2 {
			r2desc += " (whole)"
		} else {
			r2desc += fmt.Sprintf(" (from %v)", img2.Bounds())
		}
		fmt.Printf("Comparing regions %v and %v\n", r1desc, r2desc)
	}
	width := region1.Max.X - region1.Min.X
	if width != region2.Max.X - region2.Min.X { return false }
	height := region1.Max.Y - region1.Min.Y
	if height != region2.Max.Y - region2.Min.Y { return false }
	for y := 0; y < height; y++ {
		y1 := region1.Min.Y + y
		y2 := region2.Min.Y + y
		for x := 0; x < width; x++ {
			x1 := region1.Min.X + x
			x2 := region2.Min.X + x
			at1 := img1.At(x1, y1)
			at2 := img2.At(x2, y2)
			equal := ColoursAreEqual(at1, at2)
			if verbose {
				fmt.Printf("  [%-2d,%2d] (%-4d,%4d) vs (%-4d,%4d) : " +
					"%v vs %v (equal %v)\n",
					x, y, x1, y1, x2, y2, at1, at2, equal)
			}
			if !equal { return false }
		}
	}
	return true
}

func ImagesAreEqual(img1 image.Image, region1 *image.Rectangle,
	img2 image.Image, region2 *image.Rectangle,
) bool {
	return ImagesAreEqualVerbose(img1, region1, img2, region2, false)
}

func SubImage(img image.Image, region *image.Rectangle) image.Image {
	if region == nil {
		r := img.Bounds()
		region = &r
	}
	width := region.Max.X - region.Min.X
	height := region.Max.Y - region.Min.Y
	newRegion := image.Rect(0, 0, width, height)
	newImg := image.NewNRGBA(newRegion)
	for y := 0; y < height; y++ {
		y1 := region.Min.Y + y
		for x := 0; x < width; x++ {
			x1 := region.Min.X + x
			r, g, b, a := img.At(x1, y1).RGBA()
			o := newImg.PixOffset(x, y)
			if o < 0 {
				fmt.Printf("Offset of (%d, %d) in region %v is %d\n",
					x, y, newImg.Bounds(), o)
			}
			newImg.Pix[o] = uint8(r >> 8)
			newImg.Pix[o + 1] = uint8(g >> 8)
			newImg.Pix[o + 2] = uint8(b >> 8)
			newImg.Pix[o + 3] = uint8(a >> 8)
		}
	}
	return newImg
}

func SavePNG(img image.Image, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Failed to open '%s' for writing: %v", fileName, err)
	}
	if err = png.Encode(f, img); err != nil {
		return fmt.Errorf("Failed to encode PNG as '%s': %v", fileName, err)
	}
	if err = f.Close(); err != nil {
		err = fmt.Errorf("Failed to close '%s' after writing PNG: %v",
			fileName, err)
	}
	return err
}
