package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"

	"github.com/realh/repmap/pkg/repton"
)

// extractatlases scans a set of images in a directory. These images should
// be full-size screenshots from Repton Resource Pages map, typically a
// scenario's worth. There need to be enough so that between them they contain
// every possible sprite that may appear in such a map view in every colour
// scheme. The output is a directory containing an atlas for each colour. The
// sprites in each atlas will (hopefully) always be in the same order, but
// that order is undefined.

const NUM_DISTINCT_SPRITES = 33

type SpriteDefinition struct {
	image.Image
	Region image.Rectangle
	LeafName string
}

func (sd *SpriteDefinition) String() string {
	return fmt.Sprintf("%8s %v", sd.LeafName, sd.Region)
}

type ImageAndReturnChan struct {
	*SpriteDefinition
	ReturnChan chan bool
}

func NewImageAndReturnChan(sprite *SpriteDefinition) *ImageAndReturnChan {
	return &ImageAndReturnChan{
		sprite,
		make(chan bool),
	}
}

type AtlasData struct {
	OtherDataWithKnownColours map[int]*AtlasData
	DominantColour int
	// AllDistinctSprites contains all the distinct sprites contained in a set
	// of maps of one colour, including blank. There should be up to 33.
	AllDistinctSprites []*SpriteDefinition
	HasAllDistinct bool
	// ThemedSprites contains all the sprites that are unique to the theme.
	// There should be up to 27.
	ThemedSprites []*SpriteDefinition
	adderChan chan *ImageAndReturnChan
	// Once we detect a colour we can 
	forwardTo *AtlasData
}

// doAddImage checks whether this sprite is unique to the AtlasData. If it is
// it gets added. This also checks the colour and adds/merges itself in 
// OtherDataWithKnownColours accordingly. Returns true if it's a 'new' sprite.
func (ad *AtlasData) doAddImage(sprite *SpriteDefinition) bool {
	matched := false
	for _, sprt := range ad.AllDistinctSprites {
		if repton.ImagesAreEqual(sprite.Image, &sprite.Region, sprt, nil) {
			//fmt.Printf("%s matched %d\n", sprite, i)
			matched = true
			break
		}
	}
	if matched { return false }
	newImg := repton.SubImage(sprite.Image, &sprite.Region)
	newSprt := &SpriteDefinition{
		newImg,
		newImg.Bounds(),
		sprite.LeafName + "_",
	}
	ad.AllDistinctSprites = append(ad.AllDistinctSprites, newSprt)
	if len(ad.AllDistinctSprites) == NUM_DISTINCT_SPRITES {
		ad.HasAllDistinct = true
	}
	// See if we need to and can detect theme colour
	if ad.DominantColour != -1 {
		//fmt.Printf("%s: already know dominant colour %s\n",
		//	sprite, repton.ColourNames[ad.DominantColour])
		return true
	}
	colour := repton.DetectThemeOfEntireImage(newSprt, newSprt.String())
	if colour == -1 { return true }
	ad.DominantColour = colour
	other := ad.OtherDataWithKnownColours[colour]
	if other == nil {
		// This ad is the main AtlasData for colour
		ad.OtherDataWithKnownColours[colour] = ad
		return true
	}
	// This ad needs to be merged into other
	ad.forwardTo = other
	for _, sprt := range ad.AllDistinctSprites {
		if other.HasAllDistinct { break }
		other.AddImage(sprt)
	}
	if other.HasAllDistinct { ad.HasAllDistinct = true }
	ad.AllDistinctSprites = nil
	return true
}

// Initialise initialises the struct and starts a goroutine which tests whether
// each image it's fed is already present in AllDistinctSprites.
// dataWithKnownColours is a map of AtlasData keyed by the colour of each
// represented theme. An AtlasData starts with unknown (-1) DominantColour then
// adds itself to the map or combines itself with an existing one then forwards
// to it.
func (ad *AtlasData) Initialise(dataWithKnownColours map[int]*AtlasData) {
	ad.OtherDataWithKnownColours = make(map[int]*AtlasData)
	ad.DominantColour = -1
	ad.adderChan = make(chan *ImageAndReturnChan, 6)
	go func(ch chan *ImageAndReturnChan) {
		var result bool
		for !ad.HasAllDistinct {
			ic := <-ch
			if ic == nil {
				if ad.forwardTo != nil {
					ad.forwardTo.adderChan <- nil
				}
				break
			} else if ad.forwardTo != nil {
				result = ad.forwardTo.AddImage(ic.SpriteDefinition)
			} else {
				result = ad.doAddImage(ic.SpriteDefinition)
			}
			if ic != nil { ic.ReturnChan <- result }
		}
		if ad.adderChan != nil {
			close(ad.adderChan)
			ad.adderChan = nil
		}
	}(ad.adderChan)
}

// AddImage calls addImage using channels for thread-safety.
func (ad *AtlasData) AddImage(sprite *SpriteDefinition) bool {
	if ad.HasAllDistinct {
		//fmt.Printf("AddImage(%s): allDistinct, returning false\n", sprite)
		return false
	}
	imageAndReturnChan := NewImageAndReturnChan(sprite)
	ad.adderChan <- imageAndReturnChan
	result := <- imageAndReturnChan.ReturnChan
	close(imageAndReturnChan.ReturnChan)
	//fmt.Printf("AddImage(%s): !allDistinct, result %v\n", sprite, result)
	return result
}

type AtlasExtractor struct {
	DataSetsWithKnownColours map[int]*AtlasData
	Wg               *sync.WaitGroup
}

func (ae *AtlasExtractor) ProcessFile(fileName string) {
	img, error := repton.LoadImage(fileName)
	if error != nil {
		fmt.Println(error)
		return
	}
	ae.Wg.Add(1)
	ad := &AtlasData{}
	ad.Initialise(ae.DataSetsWithKnownColours)
	leafName := filepath.Base(fileName)
	go func() {
		bounds := img.Bounds()
		numColumns := bounds.Max.X - bounds.Min.X
		numRows := bounds.Max.Y - bounds.Min.Y
		for y := 0; y < numRows && !ad.HasAllDistinct; y++ {
			y0 := y * 64
			y1 := y0 + 64
			for x := 0; x < numColumns && !ad.HasAllDistinct; x++ {
				x0 := x * 64
				x1 := x0 + 64
				sprite := &SpriteDefinition{
					img, image.Rect(x0, y0, x1, y1), leafName,
				}
				ad.AddImage(sprite)
				//added := ad.AddImage(sprite)
				//var unique string
				//if added {
				//	unique = "unique"
				//} else {
				//	unique = "not unique"
				//}
				//fmt.Printf("%8s (%2d, %2d) %s\n", leafName, x, y, unique)
			}
		}
		ae.Wg.Done()
	}()
}

func (ae *AtlasExtractor) MinimumFilesNeededForCompletion() int {
	numNeeded := 6
	fmt.Print("Colours with complete set:")
	for c, d := range ae.DataSetsWithKnownColours {
		if d.HasAllDistinct {
			numNeeded--
			fmt.Printf(" %s", repton.ColourNames[c])
		}
	}
	fmt.Println()
	fmt.Print("Colours with partial set:")
	for c, d := range ae.DataSetsWithKnownColours {
		if !d.HasAllDistinct {
			fmt.Printf(" %s", repton.ColourNames[c])
		}
	}
	fmt.Println()
	fmt.Printf("Need at least %d more files\n", numNeeded)
	return numNeeded
}

func (ae *AtlasExtractor) Finish() {
	fmt.Printf("Finished with %d data sets. Incomplete data sets:\n",
		len(ae.DataSetsWithKnownColours))
	for c, d := range ae.DataSetsWithKnownColours {
		if !d.HasAllDistinct {
			fmt.Printf("  %s has %d sprites\n", repton.ColourNames[c],
				len(d.AllDistinctSprites))
		}
	}
}

func (ae *AtlasExtractor) Start(directory string) {
	repton.ProcessDirectory(directory+"/[0-9]*.png", ae, 6)
}

func (ae *AtlasExtractor) StartBatch() {
	ae.Wg = &sync.WaitGroup{}
}

func (ae *AtlasExtractor) FinishBatch() {
	fmt.Println("Finished batch, waiting for image processors")
	ae.Wg.Wait()
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("extractatlases takes 2 arguments: ")
		fmt.Println("input folder, output folder")
		os.Exit(1)
	}
	ae := AtlasExtractor{}
	ae.Start(os.Args[1])
}
