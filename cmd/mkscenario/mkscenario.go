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

func level(num int, output io.Writer) {
	// Add level number
	fmt.Fprintf(output, "%02d\n", num)
	ch := readLines(filepath.Join(os.Args[1], fmt.Sprintf("%02d.txt", num)))
	line := <-ch
	// First line of input is colour
	line = strings.ToLower(line)
	fmt.Fprintln(output, line)
	// Read all remaining lines so we can output size first
	lines := readAllLines(ch)
	// Output size
	fmt.Fprintf(output, "%d,%d\n", len(lines[0]), len(lines))
	// Output lines of level data
	for _, ln := range lines {
		fmt.Fprintln(output, ln)
	}
	// Terminate with one dash for most levels, two dashes for final level
	if num == 20 {
		fmt.Fprintln(output, "--")
	} else {
		fmt.Fprintln(output, "-")
	}
}

func transporters(output io.Writer) {
	ch := readLines(filepath.Join(os.Args[1], "Transporters.csv"))
	_ = <-ch
	// First line says "Transporters:"; we'll add their count, so
	// just like each level, read all the remaining lines first
	lines := readAllLines(ch)
	// Output size
	fmt.Fprintf(output, "Transporters: %d\n", len(lines))
	// Each line is src level, x, y, dest level, x, y
	for _, ln := range lines {
		fmt.Fprintln(output, ln)
	}
	fmt.Fprintln(output, "--")
}

func puzzle(output io.Writer) {
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
	}
	fmt.Fprintln(output, "--")
}

func main() {
	output, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("Unable to open/create %s: %s", os.Args[2], err)
	}
	for n := 1; n <= 20; n++ {
		level(n, output)
	}
	transporters(output)
	puzzle(output)
	output.Close()
}
