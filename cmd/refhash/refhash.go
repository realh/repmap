// The refhash binary takes a folder containing editor screenshots of a dummy
// level containing all possible sprites that may appear in a map.  There must
// be one file for each of the colours used by Repton, called Blue.png ...
// Red.png. The output on stdout is a JSON file containing hash values for all
// the tiles.
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/realh/repmap/pkg/edshot"
	"github.com/realh/repmap/pkg/repton"
	"github.com/realh/repmap/pkg/repton2"
)

// HashTileSet gets the hashes of a list of tile indices (corresponding to the
// T_ constants), using the positions as used in the reference level snapshots.
// bounds is the map region.
func HashTileSet(img image.Image, bounds image.Rectangle, tiles []int,
) []uint32 {
	n := len(tiles)
	positions := make([]image.Point, n)
	ch := make(chan bool, n)
	for i, t := range tiles {
		go func(i, t int) {
			var x, y int
			// Editor only allows brick ground to appear in the top row of a
			// map, so this is at 0, 0. The following rows are a copy of the
			// tile selecter, with blanks for puzzle and brick ground, and a
			// red-background skull at position (3, 5) (where row 5 is the 6th
			// row of the selecter).
			if t == repton2.T_BRICK_GROUND {
				x = 0
				y = 0
			} else {
				x = t % edshot.SEL_COLUMNS
				y = t/edshot.SEL_COLUMNS + 1
			}
			positions[i] = image.Point{x, y}
			ch <- true
		}(i, t)
	}
	// await
	for range positions {
		<-ch
	}
	return edshot.HashMapTiles(img, bounds, positions)
}

// ProcessEditorShot finds the map region in the named PNG and returns hashes of
// the tiles in it. If first is true it processes all the tiles, otherwise only
// the ones that depend on the colour theme.
func ProcessEditorShot(filename string, first bool,
) (colourThemed []uint32, anyColour []uint32) {
	fd, err := os.Open(filename)
	if err != nil {
		log.Printf("Unable to open '%s'", filename)
		return
	}
	defer fd.Close()
	img, _, err := image.Decode(fd)
	if err != nil {
		log.Printf("Unable to decode '%s' (invalid PNG?)", filename)
		return
	}
	selBounds, _, err := edshot.FindSelecters(img)
	if err != nil {
		log.Printf("Unable to find selecter tiles in '%s'", filename)
		return
	}
	mapBounds, err := edshot.FindMap(img, selBounds.Min.X,
		(selBounds.Min.Y+selBounds.Max.Y)/2)
	if err != nil {
		log.Printf("Unable to find map region in '%s'", filename)
		return
	}
	colourThemed = HashTileSet(img, mapBounds, repton2.ColourThemedTiles)
	if first {
		anyColour = HashTileSet(img, mapBounds, repton2.AnyColourTiles)
	}
	return
}

func ArrayOfUint32ToString(a []uint32) string {
	s := make([]string, len(a))
	for i, v := range a {
		s[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(s, ", ")
}

func main() {
	var anyColour string
	n := len(repton.ColourNames)
	// ch is for awaiting completed goroutines
	ch := make(chan bool, n)
	colourThemed := make([]string, n)
	for i, clr := range repton.ColourNames {
		go func(i int, clr string) {
			var ac []uint32
			ct, ac := ProcessEditorShot(
				filepath.Join(os.Args[1], clr+".png"), i == 0)
			colourThemed[i] = ArrayOfUint32ToString(ct)
			if i == 0 {
				anyColour = ArrayOfUint32ToString(ac)
			}
			ch <- true
		}(i, clr)
	}
	for range repton.ColourNames {
		<-ch
	}
	fmt.Println("{")
	fmt.Print(`  "Any": [`)
	fmt.Print(anyColour)
	fmt.Println("],")
	for i, clr := range repton.ColourNames {
		fmt.Printf(`  "%s": [`, clr)
		fmt.Print(colourThemed[i])
		if i < n-1 {
			fmt.Println("],")
		} else {
			fmt.Println("]")
		}
	}
	fmt.Println("}")
}
