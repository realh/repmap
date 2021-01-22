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
func HashTileSet(img image.Image, bounds image.Rectangle) []uint32 {
	positions := make([]image.Point, repton2.N_TILES)
	ch := make(chan bool, repton2.N_TILES)
	for i := range positions {
		go func(i int) {
			var x, y int
			// Editor only allows brick ground to appear in the top row of a
			// map, so this is at 0, 0. The following rows are a copy of the
			// tile selecter, with blanks for puzzle and brick ground, and a
			// red-background skull at position (3, 5) (where row 5 is the 6th
			// row of the selecter).
			if i == repton2.T_BRICK_GROUND {
				x = 0
				y = 0
			} else {
				x = i % edshot.SEL_COLUMNS
				y = i/edshot.SEL_COLUMNS + 1
			}
			positions[i] = image.Point{x, y}
			ch <- true
		}(i)
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
func ProcessEditorShot(filename string) []uint32 {
	img, mapBounds, _, err := edshot.LoadMap(filename)
	if err != nil {
		log.Println(err)
		return nil
	}
	return HashTileSet(img, mapBounds)
}

func ArrayOfUint32ToString(a []uint32) string {
	s := make([]string, len(a))
	for i, v := range a {
		s[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(s, ", ")
}

func main() {
	n := len(repton.ColourNames)
	// ch is for awaiting completed goroutines
	ch := make(chan bool, n)
	themedSets := make([]string, n)
	for i, clr := range repton.ColourNames {
		go func(i int, clr string) {
			ct := ProcessEditorShot(
				filepath.Join(os.Args[1], clr+".png"))
			themedSets[i] = ArrayOfUint32ToString(ct)
			ch <- true
		}(i, clr)
	}
	for range repton.ColourNames {
		<-ch
	}
	fmt.Println("{")
	for i, clr := range repton.ColourNames {
		fmt.Printf(`  "%s": [`, clr)
		fmt.Print(themedSets[i])
		if i < n-1 {
			fmt.Println("],")
		} else {
			fmt.Println("]")
		}
	}
	fmt.Println("}")
}
