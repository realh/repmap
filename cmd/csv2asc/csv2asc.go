// csv2asc converts a file from Gerald Holdsworth's CSV format to the ASCII
// format output by img2map. If there is an argument it's used as the input
// filename, otherwise stdin is used. The output is on stdout.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
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
		codes := strings.Split(line, ",")
		asc := make([]byte, len(codes))
		for i, s := range codes {
			var t byte
			if s == "unk" {
				t = repton2.T_PUZZLE
			} else {
				v, err := strconv.ParseInt(s, 10, 8)
				if err != nil {
					log.Fatalf("Can't parse '%s' as number", s)
				}
				t = byte(v)
				// Gerald's format has some different numbers
				if t == 31 {
					t = repton2.T_SKULL_RED
				} else if t == 33 {
					t = repton2.T_EGG
				} else if t == 34 {
					t = repton2.T_KEY
				} else if t == 30 {
					t = repton2.T_SAVE
				}
			}
			if t == 0 {
				t = '.'
			} else if t <= 9 {
				t += '0'
			} else {
				t += 'A' - 10
			}
			asc[i] = t
		}
		fmt.Println(string(asc))
	}
	if !errors.Is(err, io.EOF) {
		log.Fatalf("Error reading '%s': %v", os.Args[1], err)
	}
}
