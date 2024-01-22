package repton

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/crazy3lf/colorconv"
)

const (
	MATCH_WEIGHT_R = 0.299
	MATCH_WEIGHT_G = 0.587
	MATCH_WEIGHT_B = 0.114

	GOOD_MATCH = 0.001
)

func ColoursAreEqual(c1 color.Color, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// SqByteDiff returns the square of the difference between two colour words
// normalised from range (0-65535) to (0.0-1.0). The bytes are actually uint32
// with 16-bits of precision because that's what Color.RGBA returns.
func SqByteDiff(a, b uint32) float64 {
	d := a - b
	return float64(d*d) / (65535.0 * 65535.0)
}

// ColourMatch compares two colours, returning a number between 0 and 1.
// 0 means a perfect match, 1 means complete mismatch. Alpha is ignored.
func ColourMatch(c1, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()
	return math.Sqrt(SqByteDiff(r1, r2)*MATCH_WEIGHT_R +
		SqByteDiff(g1, g2)*MATCH_WEIGHT_G +
		SqByteDiff(b1, b2)*MATCH_WEIGHT_B)
}

// VerifyPixelColour checks that the pixel at (x, y) matches colour
func VerifyPixelColour(img image.Image, x, y int, colour color.Color) bool {
	return ColourMatch(img.At(x, y), colour) < GOOD_MATCH
}

// VerifyBlackEdge checks that the pixel at (x, y) matches black and the pixel
// at (x + dx, y + dy) matches grey
func VerifyBlackEdge(img image.Image, x, y, dx, dy int,
	black color.Color, grey color.Color,
) bool {
	/*
	   log.Printf(os.Stderr, "Edge: %v at %d,%d, %v at %d,%d",
	       img.At(x, y), x, y, img.At(x+dx, y+dy), x+dx, y+dy)
	*/
	return ColourMatch(img.At(x, y), black) < GOOD_MATCH &&
		ColourMatch(img.At(x+dx, y+dy), grey) < GOOD_MATCH
}

// Key colours
const (
	KC_BLUE = iota
	KC_CYAN
	KC_GREEN
	KC_MAGENTA
	KC_ORANGE
	KC_RED
	KC_BLACK
)

// Colour names
var ColourNames = [7]string{
	"Blue", "Cyan", "Green", "Magenta", "Orange", "Red", "Black",
}

// DetectColourTheme detects which of the above colours is the closest match for
// the input, returning an index into ColourNames, or -1 for no (close) match.
func DetectColourTheme(c color.Color) int {
	h, s, l := colorconv.ColorToHSL(c)
	if l < 0.0000001 { return KC_BLACK }
	if s < 0.8 { return -1 }
	if h >= 230 && h <= 250 { return KC_BLUE }
	if h <= 10 || h >= 350 { return KC_RED }
	if h >= 110 && h <= 130 { return KC_GREEN }
	if h >= 290 && h <= 310 { return KC_MAGENTA }
	if h >= 170 && h <= 190 { return KC_CYAN }
	if h >= 20 && h <= 40 { return KC_ORANGE }
	return -1
}

func CountEachColourInImageInBounds(
	img image.Image,
	bounds image.Rectangle,
) [7]int {
	var counts [7]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := img.At(x, y)
			colour := DetectColourTheme(pixel)
			if colour != -1 {
				counts[colour]++
			}
		}
	}
	return counts
}

func FindDominantColourInCounts(
	counts [7]int,
	description string,
) int {
	total := 0
	best := 0
	bestIndex := 0
	secondBest := 0
	secondBestIndex := 0
	if description != "" {
		description = fmt.Sprintf("Colour frequencies in %s:\n", description)
	}
	for i, n := range counts {
		total += n
		if description != "" {
			description += fmt.Sprintf("  %7s : %d\n",
				ColourNames[i], counts[i])
		}
		if n > secondBest && n <= best {
			secondBest = n
			secondBestIndex = i
		}
		if n > best {
			best = n
			bestIndex = i
		}
	}
	if description != "" {
		description += fmt.Sprintf("Total %d, dominant %s, second %s\n",
			total, ColourNames[bestIndex], ColourNames[secondBestIndex])
	}
	if bestIndex == KC_BLACK && secondBest != 0 {
		description += fmt.Sprintln("Black dominant, but not uniform")
		bestIndex = -1
	} else if 10*best <= 15*secondBest &&
		// blue maps tend to contain a lot of false matches for magenta
		(best == secondBest ||
			bestIndex != KC_BLUE || secondBestIndex != KC_MAGENTA) {
		if description != "" {
			description += fmt.Sprintln("Insufficient majority")
		}
		bestIndex = -1
	}
	if description != "" {
		fmt.Print(description)
	}
	return bestIndex
}

// DetectThemeOfEntireImage looks at all the pixels in an image, sorting them
// into whichever of the above colours they match best. It returns the index
// of the colour with the most matches.
func DetectThemeOfEntireImage(img image.Image, description string) int {
	numGoroutines := 4
	bounds := img.Bounds()
	//fmt.Printf("DetectThemeOfEntireImage: bounds %v\n", bounds)
	height := bounds.Max.Y - bounds.Min.Y
	rowsPerGoroutine := height / numGoroutines
	var counts [7]int
	// Each colour has its own counter goroutine
	var counterChannels [7]chan int
	for i := range counterChannels {
		counterChannels[i] = make(chan int, numGoroutines)
	}
	// Counters stop when this channel closes
	scannersDoneChannel := make(chan bool)
	// Pixel scanner goroutines use a WaitGroup to signal the other goroutines
	// that they've finished
	wg := &sync.WaitGroup{}

	// Start 4 goroutines to scan pixels
	for gr := 0; gr < numGoroutines; gr++ {
		wg.Add(1)
		go func(portion int) {
			y0 := portion * rowsPerGoroutine
			y1 := min(y0+rowsPerGoroutine, height)
			//fmt.Printf("Thread %d processing rows %d-%d\n", portion, y0, y1)
			subBounds := image.Rect(bounds.Min.X, y0, bounds.Max.X, y1)
			portionCounts := CountEachColourInImageInBounds(img, subBounds)
			for colour, count := range portionCounts {
				counterChannels[colour] <- count
			}
			wg.Done()
		}(gr)
	}

	// This goroutine waits for all the pixel scanner goroutines to finish
	// (using the WaitGroup) then closes scannersDoneChannel which signals
	// the following counter goroutines to stop
	go func() {
		wg.Wait()
		close(scannersDoneChannel)
	}()

	// Use another WaitGroup for the counter goroutines
	wg2 := sync.WaitGroup{}
	for i := 0; i < 6; i++ {
		wg2.Add(1)
		go func(colourIndex int) {
			for counting := true; counting; {
				select {
				case n := <-counterChannels[colourIndex]:
					counts[colourIndex] += n
				case <-scannersDoneChannel:
					counting = false
				}
			}
			wg2.Done()
			close(counterChannels[colourIndex])
		}(i)
	}
	wg2.Wait()
	return FindDominantColourInCounts(counts, description)
}
