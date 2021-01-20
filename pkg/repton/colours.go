package repton

import (
	"image"
	"image/color"
	"math"
)

const (
	MATCH_WEIGHT_R = 0.299
	MATCH_WEIGHT_G = 0.587
	MATCH_WEIGHT_B = 0.114

	GOOD_MATCH = 0.001
)

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
)

// Colour names
var ColourNames = [6]string{
	"Blue", "Cyan", "Green", "Magenta", "Orange", "Red",
}

// DetectColourTheme uses fuzzy logic to decide which of the above colours is
// the closest match for the input, returning an index into ColourNames or -1
// for no match.
func DetectColourTheme(c color.Color) int {
	r, g, b, _ := c.RGBA()
	if r > 0x8000 {
		if b > 0x8000 {
			return KC_MAGENTA
		} else if g > 0x4000 {
			return KC_ORANGE
		} else {
			return KC_RED
		}
	} else if g > 0x8000 {
		if b > 0x8000 {
			return KC_CYAN
		} else {
			return KC_GREEN
		}
	} else if b > 0x8000 {
		return KC_BLUE
	}
	return -1
}
