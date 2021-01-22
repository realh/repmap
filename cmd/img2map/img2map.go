// The img2map binary loads editor snapshots and outputs text files
// representing the corresponding Repton 2 maps. The input, $1, can either be a
// single PNG, a scenario folder containing 01.png ... 20.png, or a folder of
// scenario folders. $2 contains the reference tile hashes in JSON format. $3
// is the output folder; it will be filled with folders/files mirroring the
// structure below input but with each .png replaced by a .txt. If the input
// is a single file, the output folder must exist, otherwise folders will be
// created if necessary.
//
// The first line of the text file contains the colour theme eg "Blue". Each
// subsequent line is a string of characters representing a map row. '.' means
// a blank space, the next 9 tile types (in the order of the T_ constants) are
// represented by '0'-'9' and the rest by 'A'-'Z'.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/realh/repmap/pkg/edshot"
	"github.com/realh/repmap/pkg/repton"
	"github.com/realh/repmap/pkg/repton2"
)

// GetMapHashes returns hashes of the tiles in the map region.
func GetMapHashes(img image.Image, mapBounds image.Rectangle) []uint32 {
	w := (mapBounds.Max.X - mapBounds.Min.X) / edshot.MAP_TILE_WIDTH
	h := (mapBounds.Max.Y - mapBounds.Min.Y) / edshot.MAP_TILE_HEIGHT
	positions := make([]image.Point, w*h)
	ch := make(chan bool, w*h)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			go func(x, y, i int) {
				positions[i] = image.Point{x, y}
				ch <- true
			}(x, y, i)
			i++
		}
	}
	for range positions {
		<-ch
	}
	return edshot.HashMapTiles(img, mapBounds, positions)
}

// ProcessMap loads the map and works out what each tile represents using
// refTiles. It saves a text file representation of the map and returns the
// number of puzzle pieces.
func ProcessMap(inFilename, outFilename string, refTiles map[string][]uint32,
) int {
	img, mapBounds, selBounds, err := edshot.LoadMap(inFilename)
	if err != nil {
		log.Println(err)
		return 0
	}
	// What colour is this map?
	cTheme := edshot.GetMapColourTheme(img, selBounds)
	if cTheme == -1 {
		log.Printf("Failed to detect colour theme of '%s'", inFilename)
		return 0
	}
	clrName := repton.ColourNames[cTheme]
	w := (mapBounds.Max.X - mapBounds.Min.X) / edshot.MAP_TILE_WIDTH
	h := (mapBounds.Max.Y - mapBounds.Min.Y) / edshot.MAP_TILE_HEIGHT
	log.Printf("Map '%s' is %s and %d x %d", inFilename, clrName, w, h)
	// Make the hash array into a map
	refHashes := make(map[uint32]int)
	for i, h := range refTiles[clrName] {
		refHashes[h] = i
	}
	hashedTiles := GetMapHashes(img, mapBounds)
	n := len(hashedTiles)
	tValues := make([]byte, n)
	ch := make(chan bool, n)
	for i, h := range hashedTiles {
		go func(i int, h uint32) {
			t, ok := refHashes[h]
			if !ok {
				t = repton2.T_PUZZLE
			}
			var c byte
			if t == 0 {
				c = '.'
			} else if t < 10 {
				c = '0' + byte(t)
			} else {
				c = 'A' + byte(t-10)
			}
			tValues[i] = c
			ch <- ok
		}(i, h)
	}
	// await and count puzzle pieces
	nPuzzles := 0
	for range tValues {
		if !<-ch {
			nPuzzles++
		}
	}
	fd, err := os.Create(outFilename)
	if err != nil {
		log.Printf("Failed to create output file '%s': %v", outFilename, err)
		return nPuzzles
	}
	defer fd.Close()
	fmt.Fprintf(fd, "%s\n", clrName)
	for row := 0; row < n; row += w {
		fd.Write(tValues[row : row+w])
		fmt.Fprintln(fd, "")
	}
	log.Printf("%s contains %d puzzle pieces", inFilename, nPuzzles)
	return nPuzzles
}

// Returns true if filename matches "xx.png" where xx are digits, and it is a
// file
func isLevelPng(filename string) bool {
	_, leafname := filepath.Split(filename)
	if len(leafname) != 6 || !unicode.IsDigit(rune(leafname[0])) ||
		!unicode.IsDigit(rune(leafname[1])) {
		return false
	}
	if strings.HasSuffix(leafname, ".png") {
		stat, err := os.Stat(filename)
		if err != nil {
			log.Printf("Error reading '%s': %v", filename, err)
			return false
		}
		return !stat.IsDir()
	}
	return false
}

// ProcessRecursive processes the PNGs where inputRoot meets the conditions in
// the comment at the top of this file. ch should be long enough to run a
// decent number of goroutines in parallel. The result is the number of values
// you should read from ch to await all the goroutines this starts. Each value
// sent down ch is the number of puzzle pieces found in a level
func ProcessRecursive(inputRoot, outputRoot, child string,
	refTiles map[string][]uint32, ch chan int,
) int {
	var inPath, outPath string
	if child != "" {
		inPath = filepath.Join(inputRoot, child)
		outPath = filepath.Join(outputRoot, child)
	} else {
		inPath = inputRoot
		outPath = outputRoot
	}
	if isLevelPng(inPath) {
		go func() {
			outPath = outPath[:len(outPath)-3] + "txt"
			ch <- ProcessMap(inPath, outPath, refTiles)
		}()
		return 1
	}
	dir, err := os.Open(inPath)
	if err != nil {
		log.Printf("Unable to open directory '%s': %v", inPath, err)
		return 0
	}
	children, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		log.Printf("Unable to read directory '%s': %v", inPath, err)
		return 0
	}
	// Process level PNGs, but only process child directories if we're at the
	// top-level (child == "")
	numChildren := 0
	madeDir := false
	for _, c := range children {
		var subPath string
		if child == "" {
			subPath = c
		} else {
			subPath = filepath.Join(child, c)
		}
		inPath = filepath.Join(inputRoot, subPath)
		if isLevelPng(inPath) || child == "" {
			if !madeDir {
				madeDir = true
				var d string
				if child == "" {
					d = outputRoot
				} else {
					d = filepath.Join(outputRoot, child)
				}
				err := os.MkdirAll(d, 0755)
				if err != nil {
					log.Printf("Unable to create output directory '%s': %v",
						d, err)
				}
			}
			numChildren += ProcessRecursive(inputRoot, outputRoot, subPath,
				refTiles, ch)
		} else {
			log.Printf("Skipping '%s'", inPath)
		}
	}
	return numChildren
}

// LoadRefHashes loads reference hashes from a JSON file.
func LoadRefHashes(filename string) (map[string][]uint32, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	str, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}
	refs := make(map[string][]uint32)
	err = json.Unmarshal(str, &refs)
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func main() {
	if len(os.Args) != 4 {
		log.Println("img2map takes 3 arguments: ")
		log.Fatalln("input folder, output folder, reference tiles JSON")
	}
	refTiles, err := LoadRefHashes(os.Args[2])
	if err != nil {
		log.Fatalf("Failed to load/parse reference tiles: %v", err)
	}
	ch := make(chan int, 32)
	numChildren := ProcessRecursive(os.Args[1], os.Args[3], "", refTiles, ch)
	nPuzzles := 0
	for n := 0; n < numChildren; n++ {
		nPuzzles += <-ch
	}
	log.Printf("Found a total of %d puzzle pieces", nPuzzles)
}
