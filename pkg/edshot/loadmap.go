package edshot

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
)

// LoadMap loads a map, given a filename, makes an Image from it and finds the
// bounds of the tile selecter region and map region.
func LoadMap(filename string) (img image.Image,
	mapBounds image.Rectangle, selBounds image.Rectangle, e error,
) {
	fd, err := os.Open(filename)
	if err != nil {
		e = fmt.Errorf("Unable to open '%s': %v", filename, err)
		return
	}
	defer fd.Close()
	img, _, err = image.Decode(fd)
	if err != nil {
		e = fmt.Errorf("Unable to decode '%s': %v", filename, err)
		return
	}
	selBounds, _, err = FindSelecters(img)
	if err != nil {
		e = fmt.Errorf("Unable to find selecter tiles in '%s': %v",
			filename, err)
		return
	}
	mapBounds, err = FindMap(img, selBounds.Min.X,
		(selBounds.Min.Y+selBounds.Max.Y)/2)
	if err != nil {
		e = fmt.Errorf("Unable to find map region in '%s': %v", filename, err)
		return
	}
	return
}
