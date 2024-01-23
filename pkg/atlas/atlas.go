package atlas

import (
	"fmt"
	"image"
	"math"

	"github.com/realh/repmap/pkg/repton"
)

const MAX_COLUMNS = 8

// Quality returns the "quality" factor for arranging numTiles in an atlas
// with numColumns. Lower values are better.
func Quality(numTiles, numColumns int) float64 {
    wastage := numColumns - (numTiles % numColumns)
    if wastage == numColumns { wastage = 0 }
    numRows := numTiles / numColumns
    if wastage != 0 { numColumns++ }
    quality := float64(wastage) / math.Sqrt(float64(numColumns))
    quality += math.Sqrt(float64(numColumns) / float64(numRows))
    fmt.Printf("Fit factor for %d tiles in %d columns: %f\n",
        numTiles, numColumns, quality)
    return quality
}

// BestFit returns the "best" dimensions for an atlas to fit an array of
// uniform square tiles. It's a compromise between minimum wastage and
// "squareness" (a long, skinny atlas is ugly). 
func BestFit(numTiles int) (columns, rows int) {
    square := int(math.Ceil(math.Sqrt(float64(numTiles))))
    maxColumns := min(numTiles, MAX_COLUMNS)
    columns = square
    best := Quality(numTiles, columns)
    for i := square + 1; i <= maxColumns; i++ {
        quality := Quality(numTiles, i)
        if quality < best {
            best = quality
            columns = i
        }
    }
    rows = numTiles / columns
    if numTiles % columns != 0 {
        rows++
    }
    fmt.Printf("Best fit for %d tiles is %dx%d\n", numTiles, columns, rows)
    return
}

func ComposeAtlas(tiles []image.Image) image.Image {
	fmt.Printf("ComposeAtlas called with %d images\n", len(tiles))
    columns, rows := BestFit(len(tiles))
    b := tiles[0].Bounds()
    tw := b.Dx()
    th := b.Dy()
    aw := tw * columns
    ah := th * rows
    fmt.Printf("Atlas size in pixels %d x %d\n", aw, ah)
    atlas := image.NewRGBA(image.Rect(0, aw, 0, ah))
    for i, tile := range tiles {
        col := i % columns
        row := i / columns
        x0 := col * tw
        y0 := row * th
        b := image.Rect(x0, y0, x0 + tw, y0 + th)
        repton.CopyRegion(atlas, &b, tile, nil)
    }
    return atlas
}

