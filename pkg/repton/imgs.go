package repton

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func RectsAreSameSize(r1, r2 *image.Rectangle) bool {
	if r1.Dx() != r2.Dx() { return false }
	if r1.Dy() != r2.Dy() { return false }
	return true
}

func RectsAreEqual(r1, r2 *image.Rectangle) bool {
	if r1.Min.X != r2.Min.X { return false }
	if r1.Min.Y != r2.Min.Y { return false }
	if r1.Max.X != r2.Max.X { return false }
	if r1.Max.Y != r2.Max.Y { return false }
	return true
}

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
	width := region1.Dx()
	if width != region2.Dx() { return false }
	height := region1.Dy()
	if height != region2.Dy() { return false }
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
	width := region.Dx()
	height := region.Dy()
	newRegion := image.Rect(0, 0, width, height)
	newImg := image.NewRGBA(newRegion)
	CopyRegion(newImg, &newRegion, img, region)
	return newImg
}

// CopyRegion copies a region from src into dest. If either region is null
// the entire source is used. If destRegion is nil, it is set to the same size
// as srcRegion but at origin (0, 0).
func CopyRegion(dest *image.RGBA, destRegion *image.Rectangle,
	src image.Image, srcRegion *image.Rectangle,
) {
	if srcRegion == nil {
		r := src.Bounds()
		srcRegion = &r
	}
	b := dest.Bounds()
	if destRegion == nil {
		r := image.Rect(b.Min.X, b.Min.X + srcRegion.Dx(),
			b.Min.Y, b.Min.Y + srcRegion.Dy())
		destRegion = &r
	}
	if destRegion.Max.X > b.Max.X {
		destRegion.Max.X = b.Max.X
	}
	if destRegion.Max.Y > b.Max.Y {
		destRegion.Max.Y = b.Max.Y
	}
	if destRegion.Min.X < b.Min.X {
		destRegion.Min.X = b.Min.X
	}
	if destRegion.Min.Y < b.Min.Y {
		destRegion.Min.Y = b.Min.Y
	}
	
	width := destRegion.Dx()
	height := destRegion.Dy()
	for y := 0; y < height; y++ {
		y1 := destRegion.Min.Y + y
		y2 := srcRegion.Min.Y + y
		for x := 0; x < width; x++ {
			x1 := destRegion.Min.X + x
			x2 := srcRegion.Min.X + x
			r, g, b, a := src.At(x2, y2).RGBA()
			dest.SetRGBA(x1, y1, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}
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
