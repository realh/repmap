package main

import (
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
const SPRITE_SIZE = 64

var possibleDeadlocks = make(map[string]bool)
var pdMutex = sync.Mutex{}

type Lockable interface {
	Lock()
	Unlock()
}

func EnterPossibleDeadlock(s string) {
	pdMutex.Lock()
	defer pdMutex.Unlock()
	if possibleDeadlocks[s] == true {
		s = fmt.Sprintf("Multiple entry of deadlocker %s", s)
		panic(s)
	}
	possibleDeadlocks[s] = true
}

func LeavePossibleDeadlock(s string) {
	pdMutex.Lock()
	defer pdMutex.Unlock()
	if possibleDeadlocks[s] != true {
		s = fmt.Sprintf("Multiple leave of deadlocker %s", s)
		panic(s)
	}
	delete(possibleDeadlocks, s)
}

type SpriteDefinition struct {
	image.Image
	Region image.Rectangle
	LeafName string
	Verbose bool
}

func (sd *SpriteDefinition) String() string {
	return fmt.Sprintf("%8s %v", sd.LeafName, sd.Region)
}

type ImageAndReturnChan struct {
	*SpriteDefinition
	ReturnChan chan bool
}

type AtlasData struct {
	Name string
	OtherDataWithKnownColours map[int]*AtlasData
	OtherDataLock Lockable
	DominantColour int
	DominantGreens int
	// AllDistinctSprites contains all the distinct sprites contained in a set
	// of maps of one colour, including blank. There should be up to 33.
	AllDistinctSprites []*SpriteDefinition
	HasAllDistinct bool
	// ThemedSprites contains all the sprites that are unique to the theme.
	// There should be up to 27.
	ThemedSprites []*SpriteDefinition
	// Once we detect a colour we can multiplex multiple files of the same
	// colour into one AtlasData.
	forwardTo *AtlasData
	// When multiplexing we need to use a mutex on the sink.
	lock sync.Mutex
	// And addQueue helps with using the lock and threading efficiently.
	addQueue []*SpriteDefinition
	queueLock sync.Mutex
}

func (ad *AtlasData) String() string {
	c := ad.DominantColour
	var colour string
	if c != -1 {
		colour = repton.ColourNames[c]
	} else {
		colour = "unk"
	}
	var complete string
	if ad.HasAllDistinct {
		complete = "complete"
	} else {
		complete = fmt.Sprintf("%d", len(ad.AllDistinctSprites))
	}
	s := fmt.Sprintf("AD[%s, %s, %s]", ad.Name, colour, complete)
	if ad.forwardTo != nil {
		s = fmt.Sprintf("%s -> %v", s, ad.forwardTo)
	}
	return s
}

// AddImage checks whether this sprite is unique to the AtlasData. If it is
// it gets added. This also checks the colour and adds/merges itself in 
// OtherDataWithKnownColours accordingly. Returns true if it's a 'new' sprite.
func (ad *AtlasData) AddImage(sprite *SpriteDefinition) bool {
	if ad.forwardTo != nil {
		result := ad.forwardTo.AddImage(sprite)
		if ad.forwardTo.HasAllDistinct {
			ad.HasAllDistinct = true
		}
		return result
	}
	ad.queueLock.Lock()
	for _, sprt := range ad.AllDistinctSprites {
		if repton.ImagesAreEqual(
			sprite.Image, &sprite.Region, sprt, nil,
		) {
			// This sprite is already being worked on. We don't know
			// whether it will be added, and we don't want to wait, just
			//  return false.
			ad.queueLock.Unlock()
			return false
		}
	}
	ad.addQueue = append(ad.addQueue, sprite)
	ad.queueLock.Unlock()
	// If we leave the sprite in the (unlocked) addQueue until the end of
	// this function we can hold the main lock for much shorter periods
	// without potentially working on indentical sprites concurrently.
	defer func() {
		ad.queueLock.Lock()
		for i, s := range ad.addQueue {
			if s == sprite {
				ad.addQueue = slices.Delete(ad.addQueue, i, i + 1)
				break
			}
		}
		ad.queueLock.Unlock()
	}()

	matched := false
	for _, sprt := range ad.AllDistinctSprites {
		if repton.ImagesAreEqualVerbose(
			sprite.Image, &sprite.Region, sprt, nil,
			false,
			//sprite.Verbose && sprt.Verbose,
		) {
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
		sprite.Verbose,
	}
	ad.lock.Lock()
	ad.AllDistinctSprites = append(ad.AllDistinctSprites, newSprt)
	if len(ad.AllDistinctSprites) == NUM_DISTINCT_SPRITES {
		ad.HasAllDistinct = true
	}
	ad.lock.Unlock()
	//fmt.Printf("%s is unique sprite %d\n", sprite, len(ad.AllDistinctSprites))
	// See if we need to and can detect theme colour
	if ad.DominantColour != -1 {
		//fmt.Printf("Already know dominant colour of %s\n", ad)
		return true
	}
	//colour := repton.DetectThemeOfEntireImage(newSprt, newSprt.String())
	colour := repton.DetectThemeOfEntireImage(newSprt, "")
	if colour == -1 {
		//fmt.Println("Can't detect dominant colour")
		return true
	}
	// From now on it's best to hold the lock.
	ad.lock.Lock()
	defer ad.lock.Unlock()
	// Repton character and green earth (grass?) are both detected as green
	// so we can't confirm green until we have at least 3 different sprites
	if colour == repton.KC_GREEN && ad.DominantGreens < 2 {
		//fmt.Printf("Dominant colour of %s unconfirmed green\n", ad)
		ad.DominantGreens++
		return true
	}
	ad.DominantColour = colour
	other := ad.OtherDataWithKnownColours[colour]
	if other == nil {
		// This ad is the main AtlasData for colour
		fmt.Printf("%s is sink for dominant colour %s\n",
			ad, repton.ColourNames[ad.DominantColour])
		ad.OtherDataWithKnownColours[colour] = ad
		return true
	}
	// This ad needs to be merged into other
	fmt.Printf("%s forwarding to %s\n", ad, other)
	ad.forwardTo = other
	if !other.HasAllDistinct {
		for _, sprt := range ad.AllDistinctSprites {
			if other.HasAllDistinct { break }
			other.AddImage(sprt)
		}
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
func (ad *AtlasData) Initialise(
	name string,
	dataWithKnownColours map[int]*AtlasData,
	coloursDataLock Lockable,
) {
	ad.Name = name
	ad.OtherDataWithKnownColours = dataWithKnownColours
	ad.OtherDataLock = coloursDataLock
	ad.DominantColour = -1
}

type AtlasExtractor struct {
	DataSetsWithKnownColours map[int]*AtlasData
	ColoursDataLock sync.Mutex
	Wg *sync.WaitGroup
}

func (ae *AtlasExtractor) Lock() { ae.ColoursDataLock.Lock() }
func (ae *AtlasExtractor) Unlock() { ae.ColoursDataLock.Unlock() }

func (ae *AtlasExtractor) ProcessFile(fileName string) {
	dirts := []int{1, 2, 16, 26, 28, 30}
	img, error := repton.LoadImage(fileName)
	if error != nil {
		fmt.Println(error)
		return
	}
	ae.Wg.Add(1)
	fmt.Printf("ProcessFile starting on %s\n", fileName)
	ad := &AtlasData{}
	leafName := filepath.Base(fileName)
	ad.Initialise(leafName, ae.DataSetsWithKnownColours, ae)
	go func() {
		bounds := img.Bounds()
		numColumns := (bounds.Max.X - bounds.Min.X) / SPRITE_SIZE
		numRows := (bounds.Max.Y - bounds.Min.Y) / SPRITE_SIZE
		//fmt.Printf("Processing %dx%d sprites in %s\n",
		//	numColumns, numRows, fileName)
		for y := 0; y < numRows && !ad.HasAllDistinct; y++ {
			y0 := y * SPRITE_SIZE
			y1 := y0 + SPRITE_SIZE
			for x := 0; x < numColumns && !ad.HasAllDistinct; x++ {
				x0 := x * SPRITE_SIZE
				x1 := x0 + SPRITE_SIZE
				sprite := &SpriteDefinition{
					img, image.Rect(x0, y0, x1, y1), leafName,
					y == 0 && slices.Contains(dirts, x),
				}
				//fmt.Printf("Processing %s (%d,%d)\n", sprite, x, y)
				ad.AddImage(sprite)
				//added := ad.AddImage(sprite)
				//fmt.Printf("Processed %s (%d,%d), added %v, HasAll %v\n",
				//	sprite, x, y, added, ad.HasAllDistinct)
				//var unique string
				//if added {
				//	unique = "unique"
				//} else {
				//	unique = "not unique"
				//}
			}
		}
		ae.Wg.Done()
		fmt.Printf("ProcessFile finished %s\n", ad)
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
	ae.Wg.Add(1)
}

func ListDeadlocks(s string) {
	pdMutex.Lock()
	defer pdMutex.Unlock()
	if len(possibleDeadlocks) == 0 { return }
	fmt.Println(s)
	for d, b := range possibleDeadlocks {
		if !b { continue }
		fmt.Println("  ", d)
	}
	fmt.Println("****")
}

func CopyDeadlocks() map[string]bool {
	pdMutex.Lock()
	defer pdMutex.Unlock()
	m2 := make(map[string]bool)
	maps.Copy(possibleDeadlocks, m2)
	return m2
}

func DeadlocksAreEquivalent(m2 map[string]bool) bool {
	pdMutex.Lock()
	defer pdMutex.Unlock()
	if len(possibleDeadlocks) != len(m2) { return false }
	for s := range possibleDeadlocks {
		if m2[s] != true { return false }
	}
	return true
}

func (ae *AtlasExtractor) FinishBatch() {
	ae.Wg.Done()
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
	ae.DataSetsWithKnownColours = make(map[int]*AtlasData)
	ae.Start(os.Args[1])
}
