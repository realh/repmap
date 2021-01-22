// asc2csv converts a file from the ASCII format output by img2map to a CSV
// compatible with Gerald Holdsworth's utilities. If there is an argument it's
// used as the input filename, otherwise stdin is used. The output is on stdout.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/realh/repmap/pkg/repton2"
)

func main() {
	var fd *os.File
	var err error
	if len(os.Args) == 2 {
		fd, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatalf("Error opening '%s': %v", os.Args[1], err)
		}
	} else if len(os.Args) == 1 {
		fd = os.Stdin
	}
	rdr := bufio.NewReader(fd)
	var line string
	line, err = rdr.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read first line of '%s': %v", os.Args[1], err)
	}
	// In case of DOS line endings
	line = strings.TrimSpace(line)
	fmt.Println(line)
	for err == nil {
		line, err = rdr.ReadString('\n')
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			break
		}
		codes := make([]string, len(line))
		for i, c := range line {
			var t int
			if c == '.' {
				t = 0
			} else if c >= '1' && c <= '9' {
				t = int(c - '0')
			} else {
				t = int(c-'A') + 10
			}
			// Gerald's format has some different numbers
			if t == repton2.T_SKULL_RED {
				t = 31
			} else if t == repton2.T_EGG {
				t = 33
			} else if t == repton2.T_KEY {
				t = 34
			} else if t == repton2.T_SAVE {
				t = 30
			}
			if t == repton2.T_PUZZLE {
				codes[i] = "unk"
			} else {
				codes[i] = fmt.Sprintf("%d", t)
			}
		}
		fmt.Println(strings.Join(codes, ","))
	}
	if !errors.Is(err, io.EOF) {
		log.Fatalf("Error reading '%s': %v", os.Args[1], err)
	}
}
