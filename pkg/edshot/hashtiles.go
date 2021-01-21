package edshot

import "image"

// HashMapTiles generates a hash for each tile defined by map tile coordinates
// in positions in img with map region bounded by bounds.
func HashMapTiles(img image.Image, bounds image.Rectangle,
	positions []image.Point,
) (hashes []uint32) {
	n := len(positions)
	hashes = make([]uint32, n)
	ch := make(chan bool, n)
	for i, point := range positions {
		go func(i int, point image.Point) {
			b := image.Rectangle{
				Min: image.Point{point.X * MAP_TILE_WIDTH,
					point.Y * MAP_TILE_HEIGHT},
			}
			b.Min = b.Min.Add(bounds.Min)
			b.Max = b.Min.Add(image.Point{MAP_TILE_WIDTH, MAP_TILE_HEIGHT})
			hashes[i] = HashImage(img, b)
			ch <- true
		}(i, point)
	}
	// await
	for range positions {
		<-ch
	}
	return
}
