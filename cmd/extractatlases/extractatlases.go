package main

import (
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/realh/repmap/pkg/atlas"
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
	StartedFilteringSprites bool
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
	CommonSprites []*SpriteDefinition
	CommonSpritesLock sync.Mutex
	CommonSpritesWg *sync.WaitGroup
	StartedCommonSprites bool
}

func (ae *AtlasExtractor) Lock() { ae.ColoursDataLock.Lock() }
func (ae *AtlasExtractor) Unlock() { ae.ColoursDataLock.Unlock() }

func (ae *AtlasExtractor) ProcessFile(fileName string) {
	img, err := repton.LoadImage(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	leafName := filepath.Base(fileName)
	ae.Wg.Add(1)
	fmt.Printf("ProcessFile starting on %s\n", fileName)
	ad := &AtlasData{}
	ad.Initialise(leafName, ae.DataSetsWithKnownColours, ae)
	go func(ad *AtlasData) {
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
					false,
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
	}(ad)
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
	ae.Wg.Wait()
	fmt.Println("**** Finished batch ****")
	if len(ae.DataSetsWithKnownColours) < 2 {
		fmt.Println("Not enough data sets to find common sprites")
		return
	}
	ae.CommonSpritesLock.Lock()
	defer ae.CommonSpritesLock.Unlock()
	complete1 := -1
	complete2 := -1
	if !ae.StartedCommonSprites {
		for i, ad := range ae.DataSetsWithKnownColours {
			if ad.HasAllDistinct {
				if complete1 == -1 {
					complete1 = i
				} else if complete2 == -1 &&
				ad.DominantColour !=
				ae.DataSetsWithKnownColours[complete1].DominantColour {
					complete2 = i
					break
				}
			}
		}
		if complete2 == -1 {
			fmt.Println("Not enough complete sets to find common sprites")
			return
		}
		fmt.Printf("%s and %s complete, finding common sprites\n",
			ae.DataSetsWithKnownColours[complete1],
			ae.DataSetsWithKnownColours[complete2])
		ae.CommonSpritesWg = &sync.WaitGroup{}
		ae.StartedCommonSprites = true
		ae.IsolateCommonSprites(complete1, complete2)
		return
	}
	if ae.CommonSpritesWg != nil {
		fmt.Println("Waiting for previous CommonSprites job")
		ae.CommonSpritesWg.Wait()
		fmt.Printf("Identified %d common sprites\n", len(ae.CommonSprites))
		ae.CommonSpritesWg = nil
	} else {
		fmt.Println("No previous CommonSprites job")
	}
}

func (ae *AtlasExtractor) IsolateCommonSprites(i1, i2 int) {
	ae.CommonSpritesWg.Add(1)
	go func(i1, i2 int) {
		ad1 := ae.DataSetsWithKnownColours[i1]
		ad1.StartedFilteringSprites = true
		var ad2 *AtlasData
		if i2 == -1 {
			ad2 = nil
		} else {
			ad2 = ae.DataSetsWithKnownColours[i2]
			ad2.StartedFilteringSprites = true
		}
		ae.SeparateCommonSprites(ad1, ad2)
		ae.CommonSpritesWg.Done()
	}(i1, i2)
}

// SeparateCommonSprites finds sprites which are common to ad1 and ad2
// or to ad1 and ae.CommonSprites. The remaining unique sprites are copied into 
// ad1.ThemedSprites, and the same for ad2 if non-nil.
// If ad2 is nil, sprites in ad1 are tested against ae.CommonSprites, otherwise
// common sprites are copied to ae.CommonSprites.
func (ae *AtlasExtractor) SeparateCommonSprites(
	ad1 *AtlasData, ad2 *AtlasData,
) {
	var commonIn2 []int
	var ref []*SpriteDefinition
	if ad2 == nil {
		ref = ae.CommonSprites
	} else {
		ref = ad2.AllDistinctSprites
	}
	for _, s1 := range ad1.AllDistinctSprites {
		matched := false
		for i2, s2 := range ref {
			if repton.ImagesAreEqual(s1.Image, &s1.Region,
				s2.Image, &s2.Region) {
				matched = true
				if ad2 != nil {
					commonIn2 = append(commonIn2, i2)
				}
				break
			}
		}
		if matched {
			if ad2 != nil {
				ae.CommonSprites = append(ae.CommonSprites, s1)
			}
		} else {
			ad1.ThemedSprites = append(ad1.ThemedSprites, s1)
		}
	}
	if ad2 != nil {
		slices.Sort(commonIn2)
		j := 0
		k := commonIn2[j]
		for i, s := range ad2.AllDistinctSprites {
			if i == k {
				j++
				if j < len(commonIn2) {
					k = commonIn2[j]
				}
			} else {
				ad2.ThemedSprites = append(ad2.ThemedSprites, s)
			}
		}
	}
}

func SpritesToImages(sprites []*SpriteDefinition) []image.Image {
	images := make([]image.Image, len(sprites))
	for i, sprite := range sprites {
		img := sprite.Image
		region := sprite.Region
		b := sprite.Image.Bounds()
		if !repton.RectsAreEqual(&region, &b) {
			fmt.Printf("Creating new image for common sprite %d\n", i)
			img = repton.SubImage(img, &region)
		}
		images[i] = img
	}
	return images
}

func (ae *AtlasExtractor) SaveCommonSprites(fileName string) {
	imgs := SpritesToImages(ae.CommonSprites)

	dir := filepath.Dir(fileName)
	if dir == "" { dir = "." }
	for i, img := range imgs {
		fn2 := fmt.Sprintf("%s/%d.png", dir, i)
		err := repton.SavePNG(img, fn2)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}

	fmt.Printf("Calling ComposeAtlas with %d images\n", len(imgs))
	atlas := atlas.ComposeAtlas(imgs)
	err := repton.SavePNG(atlas, fileName)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
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
	ae.SaveCommonSprites(os.Args[2] + "/common.png")
}
