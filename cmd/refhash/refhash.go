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

// HashTile gets the hash of a tile in the map; bounds is the img's map region
func HashTile(img image.Image, bounds image.Rectangle, tileIndex int,
) uint32 {
	var x, y int
	if tileIndex == repton2.T_BRICK_GROUND {
		x = 0
		y = 0
	} else {
		x = tileIndex % edshot.SEL_COLUMNS
		y = tileIndex/edshot.SEL_COLUMNS + 1
	}
	bounds.Min.X += x * edshot.MAP_TILE_WIDTH
	bounds.Min.Y += y * edshot.MAP_TILE_HEIGHT
	bounds.Max.X = bounds.Min.X + edshot.MAP_TILE_WIDTH
	bounds.Max.Y = bounds.Min.Y + edshot.MAP_TILE_HEIGHT
	return edshot.HashImage(img, bounds)
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
	n := len(repton2.ColourThemedTiles)
	colourThemed = make([]uint32, n)
	if first {
		l := len(repton2.AnyColourTiles)
		anyColour = make([]uint32, l)
		n += l
	}
	// ch is for awaiting completed goroutines
	ch := make(chan bool, len(colourThemed)+len(anyColour))
	for i, t := range repton2.ColourThemedTiles {
		go func(i int, t int) {
			colourThemed[i] = HashTile(img, mapBounds, t)
			ch <- true
		}(i, t)
	}
	if first {
		for i, t := range repton2.AnyColourTiles {
			go func(i int, t int) {
				anyColour[i] = HashTile(img, mapBounds, t)
				ch <- true
			}(i, t)
		}
	}
	// await
	for i := 0; i < n; i++ {
		<-ch
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
