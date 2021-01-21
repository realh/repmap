// edshot contains functions etc for analysing screenshots of the editor
package edshot

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/realh/repmap/pkg/repton"
)

const (
	SEL_TILE_WIDTH         = 32
	SEL_TILE_HEIGHT        = 32
	SEL_TILE_BORDER        = 1
	PADDED_SEL_TILE_WIDTH  = SEL_TILE_WIDTH + 2*SEL_TILE_BORDER
	PADDED_SEL_TILE_HEIGHT = SEL_TILE_HEIGHT + 2*SEL_TILE_BORDER
	SEL_COLUMNS            = 6
	MAP_TILE_WIDTH         = 16
	MAP_TILE_HEIGHT        = 15
)

// FindSelecters finds the extremities of the tile selecter area and also
// returns the colour of the window background. This is fairly easy because the
// region has a 1px black border. Each value in rect is the coordinate that the
// border falls on.
func FindSelecters(img image.Image) (rect image.Rectangle,
	grey color.Color, err error,
) {
	bounds := img.Bounds()
	// First look for the right border halfway down the window
	y := (bounds.Max.Y - bounds.Min.Y) / 2
	black := color.RGBA{0, 0, 0, 255}
	var x int
	// 32 * 8 is < minimum space needed to display actual map.
	// 16 allows for window border.
	for x = bounds.Max.X - 16; x > 32*8; x-- {
		px := img.At(x, y)
		if repton.ColourMatch(px, black) < repton.GOOD_MATCH {
			break
		} else {
			grey = px
		}
	}
	if x <= 32*8 {
		err = fmt.Errorf("Right edge of selecter region not found")
		return
	}
	rect.Max.X = x
	rect.Min.X = x - PADDED_SEL_TILE_WIDTH*SEL_COLUMNS + SEL_TILE_BORDER
	// Verify the left edge
	if !repton.VerifyBlackEdge(img, rect.Min.X, y, -1, 0, black, grey) {
		return rect, grey, fmt.Errorf("Left edge of selecter region not found")
	}
	// From here find the top edge; 100 is arbitrary; x is still right edge
	// because left may have the white dotted outline cursor in the way
	for ; y > 100; y-- {
		px := img.At(x, y)
		if repton.ColourMatch(px, black) > repton.GOOD_MATCH {
			break
		}
	}
	if y <= 100 {
		err = fmt.Errorf("Top edge of selecter region not found")
		return
	}
	// Make sure this is a valid edge
	y++
	rect.Min.Y = y
	if !repton.VerifyBlackEdge(img, x, y, 0, -1, black, grey) {
		err = fmt.Errorf("Top edge of selecter region not found")
		return
	}
	y += 34*6 - 1
	rect.Max.Y = y
	if !repton.VerifyBlackEdge(img, x, y, 0, 1, black, grey) {
		err = fmt.Errorf("Bottom edge of selecter region not found")
		return
	}
	return rect, grey, nil
}

// FindMapRow finds the extremities of the map portion of the snapshot. x and
// y should be on the left edge of the selecter area, approximately halfway
// down.  The result has inclusive Min and exclusive Max.
func FindMapRow(img image.Image, x, y int, grey color.Color,
) (minX, maxX int, err error) {
	x--
	// Somewhere left of that we should encounter a whitish plinth border
	for ; x >= 0; x-- {
		if repton.ColourMatch(img.At(x, y), grey) > repton.GOOD_MATCH {
			break
		}
	}
	if x <= 0 {
		err = fmt.Errorf("Left border of selecter plinth not found")
		return
	}
	// Border may be > 1 pixel wide
	for ; x >= 0; x-- {
		if repton.ColourMatch(img.At(x, y), grey) < repton.GOOD_MATCH {
			break
		}
	}
	if x <= 0 {
		err = fmt.Errorf("Didn't encounter more grey left of selecter plinth")
		return
	}
	// After more grey we should find a trench
	for ; x >= 0; x-- {
		if repton.ColourMatch(img.At(x, y), grey) > repton.GOOD_MATCH {
			break
		}
	}
	if x <= 0 {
		err = fmt.Errorf("Background left of plinth extended to left of window")
		return
	}
	// Trench is > 1 pixel wide
	for ; x >= 0; x-- {
		if repton.ColourMatch(img.At(x, y), grey) < repton.GOOD_MATCH {
			break
		}
	}
	if x <= 0 {
		err = fmt.Errorf("No background within map border")
		return
	}
	// The next non-grey we encounter should be the edge of the map
	for ; x >= 0; x-- {
		if repton.ColourMatch(img.At(x, y), grey) > repton.GOOD_MATCH {
			break
		}
	}
	if x <= 0 {
		err = fmt.Errorf("Main background extended to left of window")
		return
	}
	maxX = x + 1
	// From here subtract MAP_TILE_WIDTH pixels at a time, looking for grey
	for ; x >= 0; x -= MAP_TILE_WIDTH {
		if repton.ColourMatch(img.At(x, y), grey) < repton.GOOD_MATCH {
			break
		}
	}
	if x < 0 {
		err = fmt.Errorf("Couldn't find left edge of map")
		return
	}
	x++
	minX = x
	return
}

func FindMapTopAndBottom(img image.Image, x, y int, grey color.Color,
) (minY, maxY int, err error) {
	// Now start looking up for top edge; better to use right edge just in case
	// something drastic's happened to the "Level n" label
	y0 := y
	for ; y > 0; y-- {
		if repton.ColourMatch(img.At(x, y), grey) < repton.GOOD_MATCH {
			break
		}
	}
	if y <= 0 {
		err = fmt.Errorf("Couldn't find top edge of map")
		return
	}
	y++
	minY = y
	// Don't need to start checking for grey again until we're below original y
	for ; y <= y0; y += MAP_TILE_HEIGHT {
	}
	for ; y <= y0+32*MAP_TILE_HEIGHT; y += MAP_TILE_HEIGHT {
		if repton.ColourMatch(img.At(x, y), grey) < repton.GOOD_MATCH {
			break
		}
	}
	if y > y0+32*MAP_TILE_HEIGHT {
		err = fmt.Errorf("Couldn't find bottom edge of map")
		return
	}
	maxY = y
	return
}

// FindMap finds the extremities of the map portion of the snapshot. x and y
// should be on the left edge of the selecter area, approximately halfway down.
// The result has inclusive Min and exclusive Max.
func FindMap(img image.Image, x, y int) (rect image.Rectangle, err error) {
	// Immediately left of the selecter is a verified grey region
	x--
	grey := img.At(x, y)
	found := false
	for n := 0; n < 20; n++ {
		minX, maxX, err := FindMapRow(img, x, y+n, grey)
		if err != nil {
			log.Printf("FindMapRow failed at row %d: %v", y+n, err)
			continue
		}
		minY1, maxY1, err := FindMapTopAndBottom(img, minX, y+n, grey)
		if err != nil {
			log.Printf("FindMapTopAndBottom failed for minX at row %d: %v",
				y+n, err)
			continue
		}
		minY2, maxY2, err := FindMapTopAndBottom(img, maxX-1, y+n, grey)
		if err != nil {
			log.Printf("FindMapTopAndBottom failed for maxX at row %d): %v",
				y+n, err)
			continue
		}
		if minY1 != minY2 || maxY1 != maxY2 {
			log.Printf("FindMapTopAndBottom mismatch at row %d: (%d,%d) vs (%d,%d)",
				y+n, minY1, maxY1, minY2, maxY2)
			continue
		}
		found = true
		rect.Min.X = minX
		rect.Max.X = maxX
		rect.Min.Y = minY1
		rect.Max.Y = maxY1
		break
	}
	if !found {
		err = fmt.Errorf("Completely failed to find map")
	}
	return
}

// GetMapColourTheme samples a particular pixel from the selecter region
// (r) to determine the map's colour theme
func GetMapColourTheme(img image.Image, r image.Rectangle) int {
	return repton.DetectColourTheme(img.At(r.Min.X+5*PADDED_SEL_TILE_WIDTH+9,
		r.Min.Y+9))
}
