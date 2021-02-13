// mkscenario takes a folder ($1) full of text files output by img2map, plus
// Puzzle.csv and Transporters.csv and compiles them into one big file ($2)
// which is easier to manage in an Apple bundle.
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PosKey encodes a map level number and x, y coords in a single int
type PosKey int

// NewPosKey creates a new PosKey from the given level and position
func NewPosKey(level, x, y int) PosKey {
	return PosKey((level << 16) | (x << 8) | y)
}

// Decode returns the individual components of a PosKey
func (pk PosKey) Decode() (level, x, y int) {
	i := int(pk)
	level = i >> 16
	x = (i >> 8) & 0xff
	y = i & 0xff
	return
}

func (pk PosKey) String() string {
	l, x, y := pk.Decode()
	return fmt.Sprintf("%d,%d,%d", l, x, y)
}

// readLines reads a text file one (trimmed) line at a time
func readLines(filename string) <-chan string {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Couldn't load file %s: %s", filename, err)
	}
	rdr := bufio.NewReader(fd)
	ch := make(chan string)
	go func() {
		for {
			line, err := rdr.ReadString('\n')
			if len(line) != 0 {
				line = strings.TrimSpace(line)
				ch <- line
			}
			if err != nil && err != io.EOF {
				log.Fatalf("Couldn't read from file %s: %s", filename, err)
			}
			if err != nil || len(line) == 0 {
				break
			}
		}
		close(ch)
		fd.Close()
	}()
	return ch
}

// borders holds the lines from the Borders.csv file, one per level.
// The format is Top,Map,Tile where:
// Top = Underground, Surface or Meteors
// Map = Viewable, Visited or No
// Tile = Index code of tile which surrounds the map
var borders []string

// tiles holds ASCII codes of all tiles in the scenario that we need to check:
// blank, transporter and puzzle
var tiles = make(map[PosKey]rune)

// puzzles holds locations all puzzle pieces found in Puzzle.csv
var puzzles = make(map[PosKey]int)

// transporters holds all transporters found in Transporters.csv
// Key is src, val is dest
var transporters = make(map[PosKey]PosKey)

func readAllLines(ch <-chan string) []string {
	var lines []string
	for {
		line := <-ch
		if len(line) == 0 {
			break
		}
		lines = append(lines, line)
	}
	return lines
}

func doLevel(num int, output io.Writer) {
	// Add level number
	fmt.Fprintf(output, "%02d\n", num)
	ch := readLines(filepath.Join(os.Args[1], fmt.Sprintf("%02d.txt", num)))
	line := <-ch
	// Output borders info
	fmt.Fprintln(output, borders[num-1])
	// First line of input is colour
	line = strings.ToLower(line)
	fmt.Fprintln(output, line)
	// Read all remaining lines so we can output size first
	lines := readAllLines(ch)
	// Output size
	fmt.Fprintf(output, "%d,%d\n", len(lines[0]), len(lines))
	// Output lines of level data
	for y, ln := range lines {
		fmt.Fprintln(output, ln)
		for x, c := range ln {
			switch c {
			case '.', 'O', 'U':
				pk := NewPosKey(num, x, y)
				tiles[pk] = c
			}
		}
	}
	// Terminate with one dash for most levels, two dashes for final level
	if num == 20 {
		fmt.Fprintln(output, "--")
	} else {
		fmt.Fprintln(output, "-")
	}
}

func doTransporters(output io.Writer) {
	ch := readLines(filepath.Join(os.Args[1], "Transporters.csv"))
	_ = <-ch
	// First line says "Transporters:"; we'll add their count, so
	// just like each level, read all the remaining lines first
	lines := readAllLines(ch)
	// Output size
	fmt.Fprintf(output, "Transporters: %d\n", len(lines))
	// Each line is src level, x, y, dest level, x, y
	for n, ln := range lines {
		fmt.Fprintln(output, ln)
		svals := strings.Split(lines[n], ",")
		ivals := make([]int, len(svals))
		var err error
		if len(ivals) != 6 {
			err = fmt.Errorf("Not 6 fields")
		}
		if err == nil {
			for n, s := range svals {
				var i int64
				i, err = strconv.ParseInt(s, 10, 64)
				if err != nil {
					log.Printf(
						"Unable to parse transporter from line %d: %s: %s",
						n, ln, err)
					break
				}
				ivals[n] = int(i)
			}
		}
		if err == nil {
			transporters[NewPosKey(ivals[0], ivals[1], ivals[2])] =
				NewPosKey(ivals[3], ivals[4], ivals[5])
		}
	}
	fmt.Fprintln(output, "--")
}

func doPuzzle(output io.Writer) {
	ch := readLines(filepath.Join(os.Args[1], "Puzzle.csv"))
	line := <-ch
	// First line contains width, height (number of tiles)
	size := strings.Split(line, ",")
	var w, h int64
	var err error
	w, err = strconv.ParseInt(size[0], 10, 64)
	if err == nil {
		h, err = strconv.ParseInt(size[1], 10, 64)
	}
	if err != nil {
		log.Fatalf("Couldn't parse puzzle size from %s: %s", line, err)
	}
	// Output size
	fmt.Fprintf(output, "Puzzle: %d,%d\n", w, h)
	count := int(w * h)
	// Last line seems to be duplicate, so just read all lines, and process
	// the number we calculated, ignoring surplus
	lines := readAllLines(ch)
	// Each line is level, x, y
	for n := 0; n < count; n++ {
		fmt.Fprintln(output, lines[n])
		vals := strings.Split(lines[n], ",")
		var l, x, y int64
		l, err = strconv.ParseInt(vals[0], 10, 64)
		if err == nil {
			x, err = strconv.ParseInt(vals[1], 10, 64)
		}
		if err == nil {
			y, err = strconv.ParseInt(vals[1], 10, 64)
		}
		if err != nil {
			log.Printf("Couldn't parse puzzle position from line %d: %s: %s",
				n, lines[n], err)
		} else {
			puzzles[NewPosKey(int(l), int(x), int(y))] = n
		}
	}
	fmt.Fprintln(output, "--")
}

func validate() {
	for pk, code := range tiles {
		switch code {
		case 'O':
			if _, ok := transporters[pk]; !ok {
				log.Printf(
					"Tile at %s is a transporter not found in Transporters.csv",
					pk)
			}
		case 'U':
			if _, ok := puzzles[pk]; !ok {
				log.Printf(
					"Tile at %s is a puzzle piece not found in Puzzle.csv",
					pk)
			}
		}
	}
	for pk := range puzzles {
		if tiles[pk] != 'U' {
			log.Printf(
				"Puzzles.csv contains %s, but tile is not a puzzle piece", pk)
		}
	}
	for src, dest := range transporters {
		if tiles[src] != 'O' {
			log.Printf(
				"Transporters.csv contains src %s, but tile is not a tp",
				src)
		}
		if tiles[dest] != '.' {
			log.Printf(
				"Transporters.csv contains dest %s, but tile is not a blank",
				dest)
		}
	}
}

func main() {
	output, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("Unable to open/create %s: %s", os.Args[2], err)
	}
	ch := readLines(filepath.Join(os.Args[1], "Borders.csv"))
	borders = readAllLines(ch)
	for n := 1; n <= 20; n++ {
		doLevel(n, output)
	}
	doTransporters(output)
	doPuzzle(output)
	output.Close()
}
