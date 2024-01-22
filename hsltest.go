package main

import (
	"fmt"
	"image/color"

	"github.com/crazy3lf/colorconv"
)

func showHsl(name string, c color.Color) {
    r, g, b, _ := c.RGBA()
    h, s, l := colorconv.ColorToHSL(c)
    fmt.Printf("%8s : #%02x%02x%02x : hsl(%f, %f, %f)\n",
        name, r >> 8, g >> 8, b >> 8, h, s, l)
}

func main() {
    showHsl("red", color.RGBA{255, 0, 0, 255})
    showHsl("green", color.RGBA{0, 255, 0, 255})
    showHsl("blue", color.RGBA{0, 0, 255, 255})
    showHsl("magenta", color.RGBA{255, 0, 255, 255})
    showHsl("cyan", color.RGBA{0, 255, 255, 255})
    showHsl("orange", color.RGBA{255, 127, 0, 255})
}
